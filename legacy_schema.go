package tfsdk

import (
	"fmt"

	"github.com/apparentlymart/terraform-sdk/tflegacy"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/zclconf/go-cty/cty"
)

// prepareLegacySchema converts a subset of the information in the given legacy
// schema map to the new schema representation. It converts only the minimal
// information required to support the shimming to the old API and to support
// returning an even smaller subset of the schema to Terraform Core when
// requested.
func prepareLegacySchema(old map[string]*tflegacy.Schema, enableAsSingle bool) *tfschema.BlockType {
	ret := &tfschema.BlockType{
		Attributes:       map[string]*tfschema.Attribute{},
		NestedBlockTypes: map[string]*tfschema.NestedBlockType{},
	}

	for name, schema := range old {
		if schema.Elem == nil {
			ret.Attributes[name] = prepareLegacySchemaAttribute(schema, enableAsSingle)
			continue
		}
		if schema.Type == tflegacy.TypeMap {
			// For TypeMap in particular, it isn't valid for Elem to be a
			// *Resource (since that would be ambiguous in flatmap) and
			// so Elem is treated as a TypeString schema if so. This matches
			// how the field readers treat this situation, for compatibility
			// with configurations targeting Terraform 0.11 and earlier.
			if _, isResource := schema.Elem.(*tflegacy.Resource); isResource {
				sch := *schema // shallow copy
				sch.Elem = &tflegacy.Schema{
					Type: tflegacy.TypeString,
				}
				ret.Attributes[name] = prepareLegacySchemaAttribute(&sch, enableAsSingle)
				continue
			}
		}
		switch schema.ConfigMode {
		case tflegacy.SchemaConfigModeAttr:
			ret.Attributes[name] = prepareLegacySchemaAttribute(schema, enableAsSingle)
		case tflegacy.SchemaConfigModeBlock:
			ret.NestedBlockTypes[name] = prepareLegacySchemaNestedBlockType(schema, enableAsSingle)
		default: // SchemaConfigModeAuto, or any other invalid value
			if schema.Computed && !schema.Optional {
				// Computed-only schemas are always handled as attributes,
				// because they never appear in configuration.
				ret.Attributes[name] = prepareLegacySchemaAttribute(schema, enableAsSingle)
				continue
			}
			switch schema.Elem.(type) {
			case *tflegacy.Schema, tflegacy.ValueType:
				ret.Attributes[name] = prepareLegacySchemaAttribute(schema, enableAsSingle)
			case *tflegacy.Resource:
				ret.NestedBlockTypes[name] = prepareLegacySchemaNestedBlockType(schema, enableAsSingle)
			default:
				// Should never happen for a valid schema
				panic(fmt.Errorf("invalid Schema.Elem %#v; need *Schema or *Resource", schema.Elem))
			}
		}
	}

	return ret
}

func prepareLegacySchemaAttribute(legacy *tflegacy.Schema, enableAsSingle bool) *tfschema.Attribute {
	// The Schema.DefaultFunc capability adds some extra weirdness here since
	// it can be combined with "Required: true" to create a sitution where
	// required-ness is conditional. Terraform Core doesn't share this concept,
	// so we must sniff for this possibility here and conditionally turn
	// off the "Required" flag if it looks like the DefaultFunc is going
	// to provide a value.
	// This is not 100% true to the original interface of DefaultFunc but
	// works well enough for the EnvDefaultFunc and MultiEnvDefaultFunc
	// situations, which are the main cases we care about.
	//
	// Note that this also has a consequence for commands that return schema
	// information for documentation purposes: running those for certain
	// providers will produce different results depending on which environment
	// variables are set. We accept that weirdness in order to keep this
	// interface to core otherwise simple.
	reqd := legacy.Required
	opt := legacy.Optional
	if reqd && legacy.DefaultFunc != nil {
		v, err := legacy.DefaultFunc()
		// We can't report errors from here, so we'll instead just force
		// "Required" to false and let the provider try calling its
		// DefaultFunc again during the validate step, where it can then
		// return the error.
		if err != nil || (err == nil && v != nil) {
			reqd = false
			opt = true
		}
	}

	return &tfschema.Attribute{
		Type:        prepareLegacySchemaType(legacy, enableAsSingle),
		Optional:    opt,
		Required:    reqd,
		Computed:    legacy.Computed,
		Sensitive:   legacy.Sensitive,
		Description: legacy.Description,
	}
}

func prepareLegacySchemaType(legacy *tflegacy.Schema, enableAsSingle bool) cty.Type {
	switch legacy.Type {
	case tflegacy.TypeString:
		return cty.String
	case tflegacy.TypeBool:
		return cty.Bool
	case tflegacy.TypeInt, tflegacy.TypeFloat:
		// configschema doesn't distinguish int and float, so helper/schema
		// will deal with this as an additional validation step after
		// configuration has been parsed and decoded.
		return cty.Number
	case tflegacy.TypeList, tflegacy.TypeSet, tflegacy.TypeMap:
		var elemType cty.Type
		switch set := legacy.Elem.(type) {
		case *tflegacy.Schema:
			elemType = prepareLegacySchemaType(set, enableAsSingle)
		case tflegacy.ValueType:
			// This represents a mistake in the provider code, but it's a
			// common one so we'll just shim it.
			elemType = prepareLegacySchemaType(&tflegacy.Schema{Type: set}, enableAsSingle)
		case *tflegacy.Resource:
			// By default we construct a NestedBlock in this case, but this
			// behavior is selected either for computed-only schemas or
			// when ConfigMode is explicitly SchemaConfigModeBlock.
			// See schemaMap.CoreConfigSchema for the exact rules.
			elemType = prepareLegacySchema(set.Schema, enableAsSingle).ImpliedCtyType()
		default:
			if set != nil {
				// Should never happen for a valid schema
				panic(fmt.Errorf("invalid Schema.Elem %#v; need *Schema or *Resource", legacy.Elem))
			}
			// Some pre-existing schemas assume string as default, so we need
			// to be compatible with them.
			elemType = cty.String
		}
		if legacy.AsSingle && enableAsSingle {
			// In AsSingle mode, we artifically force a TypeList or TypeSet
			// attribute in the SDK to be treated as a single value by Terraform Core.
			// This must then be fixed up in the shim code (in helper/plugin) so
			// that the SDK still sees the lists or sets it's expecting.
			return elemType
		}
		switch legacy.Type {
		case tflegacy.TypeList:
			return cty.List(elemType)
		case tflegacy.TypeSet:
			return cty.Set(elemType)
		case tflegacy.TypeMap:
			return cty.Map(elemType)
		default:
			// can never get here in practice, due to the case we're inside
			panic("invalid collection type")
		}
	default:
		// should never happen for a valid schema
		panic(fmt.Errorf("invalid Schema.Type %s", legacy.Type))
	}
}

func prepareLegacySchemaNestedBlockType(legacy *tflegacy.Schema, enableAsSingle bool) *tfschema.NestedBlockType {
	ret := &tfschema.NestedBlockType{}
	if nested := prepareLegacySchema(legacy.Elem.(*tflegacy.Resource).Schema, enableAsSingle); nested != nil {
		ret.Content = *nested
	}
	switch legacy.Type {
	case tflegacy.TypeList:
		ret.Nesting = tfschema.NestingList
	case tflegacy.TypeSet:
		ret.Nesting = tfschema.NestingSet
	case tflegacy.TypeMap:
		ret.Nesting = tfschema.NestingMap
	default:
		// Should never happen for a valid schema
		panic(fmt.Errorf("invalid s.Type %s for s.Elem being resource", legacy.Type))
	}

	ret.MinItems = legacy.MinItems
	ret.MaxItems = legacy.MaxItems

	if legacy.AsSingle && enableAsSingle {
		// In AsSingle mode, we artifically force a TypeList or TypeSet
		// attribute in the SDK to be treated as a single block by Terraform Core.
		// This must then be fixed up in the shim code (in helper/plugin) so
		// that the SDK still sees the lists or sets it's expecting.
		ret.Nesting = tfschema.NestingSingle
	}

	if legacy.Required && legacy.MinItems == 0 {
		// new schema doesn't have a "required" representation for nested
		// blocks, but we can fake it by requiring at least one item.
		ret.MinItems = 1
	}
	if legacy.Optional && legacy.MinItems > 0 {
		// Historically helper/schema would ignore MinItems if Optional were
		// set, so we must mimic this behavior here to ensure that providers
		// relying on that undocumented behavior can continue to operate as
		// they did before.
		ret.MinItems = 0
	}
	if legacy.Computed && !legacy.Optional {
		// MinItems/MaxItems are meaningless for computed nested blocks, since
		// they are never set by the user anyway. This ensures that we'll never
		// generate weird errors about them.
		ret.MinItems = 0
		ret.MaxItems = 0
	}

	return ret
}

func prepareLegacyResourceTypeSchema(legacy *tflegacy.Resource, shimmed bool) *tfschema.BlockType {
	enableAsSingle := !shimmed
	block := prepareLegacySchema(legacy.Schema, enableAsSingle)

	if block.Attributes == nil {
		block.Attributes = map[string]*tfschema.Attribute{}
	}

	// Add the implicitly required "id" field if it doesn't exist
	if block.Attributes["id"] == nil {
		block.Attributes["id"] = &tfschema.Attribute{
			Type:     cty.String,
			Optional: true,
			Computed: true,
		}
	}

	_, timeoutsAttr := block.Attributes[tflegacy.TimeoutsConfigKey]
	_, timeoutsBlock := block.NestedBlockTypes[tflegacy.TimeoutsConfigKey]

	// Insert configured timeout values into the schema, as long as the schema
	// didn't define anything else by that name.
	if legacy.Timeouts != nil && !timeoutsAttr && !timeoutsBlock {
		timeouts := tfschema.BlockType{
			Attributes: map[string]*tfschema.Attribute{},
		}

		if legacy.Timeouts.Create != nil {
			timeouts.Attributes[tflegacy.TimeoutCreate] = &tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			}
		}

		if legacy.Timeouts.Read != nil {
			timeouts.Attributes[tflegacy.TimeoutRead] = &tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			}
		}

		if legacy.Timeouts.Update != nil {
			timeouts.Attributes[tflegacy.TimeoutUpdate] = &tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			}
		}

		if legacy.Timeouts.Delete != nil {
			timeouts.Attributes[tflegacy.TimeoutDelete] = &tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			}
		}

		if legacy.Timeouts.Default != nil {
			timeouts.Attributes[tflegacy.TimeoutDefault] = &tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			}
		}

		block.NestedBlockTypes[tflegacy.TimeoutsConfigKey] = &tfschema.NestedBlockType{
			Nesting: tfschema.NestingSingle,
			Content: timeouts,
		}
	}

	return block
}
