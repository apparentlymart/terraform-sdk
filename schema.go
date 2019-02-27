package tfsdk

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

type SchemaBlockType struct {
	Attributes       map[string]*SchemaAttribute
	NestedBlockTypes map[string]*SchemaNestedBlockType
}

type SchemaAttribute struct {
	// Type defines the Terraform Language type that is required for values of
	// this attribute. Set Type to cty.DynamicPseudoType to indicate that any
	// type is allowed. The ValidateFunc field can be used to provide more
	// specific constraints on acceptable values.
	Type cty.Type

	// Required, Optional, and Computed together define how this attribute
	// behaves in configuration and during change actions.
	//
	// Required and Optional are mutually exclusive. If Required is set then
	// a value for the attribute must always be provided as an argument in
	// the configuration. If Optional is set then the configuration may omit
	// definition of the attribute, causing it to be set to a null value.
	// Optional can also be used in conjunction with computed, as described
	// below.
	//
	// Set Computed to indicate that the provider itself decides the value for
	// the attribute. When Computed is used in isolation, the attribute may not
	// be used as an argument in configuration at all. When Computed is combined
	// with Optional, the attribute may optionally be defined in configuration
	// but the provider supplies a default value when it is not set.
	//
	// Required may not be used in combination with either Optional or Computed.
	Required, Optional, Computed bool

	// Sensitive is a request to protect values of this attribute from casual
	// display in the default Terraform UI. It may also be used in future for
	// more complex propagation of derived sensitive values. Set this flag
	// for any attribute that may contain passwords, private keys, etc.
	Sensitive bool

	// Description is an English language description of the meaning of values
	// of this attribute, written as at least one full sentence with a leading
	// capital letter and trailing period. Use multiple full sentences if any
	// clarifying remarks are needed, but try to keep descriptions consise.
	Description string

	// ValidateFn, if non-nil, must be set to a function that takes a single
	// argument and returns Diagnostics. The function will be called during
	// validation and passed a representation of the attribute value converted
	// to the type of the function argument using package gocty.
	//
	// If a given value cannot be converted to the first argument type, the
	// function will not be called and instead a generic type-related error
	// will be returned automatically to the user. If the given function has
	// the wrong number of arguments or an incorrect return value, validation
	// will fail with an error indicating a bug in the provider.
	//
	// Diagnostics returned from the function must have Path values relative
	// to the given value, which will be appended to the base path by the
	// caller during a full validation walk. For primitive values (which have
	// no elements or attributes), set Path to nil.
	ValidateFn interface{}
}

type SchemaNestedBlockType struct {
	Nesting SchemaNestingMode
	Content SchemaBlockType

	MaxItems, MinItems int
}

type SchemaNestingMode int

const (
	schemaNestingInvalid SchemaNestingMode = iota
	SchemaNestingSingle
	SchemaNestingList
	SchemaNestingMap
	SchemaNestingSet
)

// Validate checks that the given object value is suitable for the recieving
// block type, returning diagnostics if not.
func (b *SchemaBlockType) Validate(val cty.Value) Diagnostics {
	var diags Diagnostics
	if !val.Type().IsObjectType() {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid block object",
			Detail:   "An object value is required to represent this block.",
		})
		return diags
	}

	// Capacity 3 here is so that we have room for a nested block type, an
	// index, and a nested attribute name without allocating more. Each loop
	// below will mutate this backing array but not the original empty slice.
	path := make(cty.Path, 0, 3)

	for name, attrS := range b.Attributes {
		path := path.GetAttr(name)
		av := val.GetAttr(name)
		attrDiags := attrS.Validate(av)
		diags = diags.Append(attrDiags.UnderPath(path))
	}

	for name, blockS := range b.NestedBlockTypes {
		path := path.GetAttr(name)
		av := val.GetAttr(name)

		switch blockS.Nesting {
		case SchemaNestingSingle:
			blockDiags := blockS.Content.Validate(av)
			diags = diags.Append(blockDiags.UnderPath(path))
		case SchemaNestingList, SchemaNestingMap:
			for it := av.ElementIterator(); it.Next(); {
				ek, ev := it.Element()
				path := path.Index(ek)
				blockDiags := blockS.Content.Validate(ev)
				diags = diags.Append(blockDiags.UnderPath(path))
			}
		case SchemaNestingSet:
			// We handle sets separately because we can't describe a path
			// through a set element (it has no key to use) and so any errors
			// in a set block are indicated at the set itself. Nested blocks
			// backed by sets are fraught with oddities like these, so providers
			// should avoid using them except for historical compatibilty.
			for it := av.ElementIterator(); it.Next(); {
				_, ev := it.Element()
				blockDiags := blockS.Content.Validate(ev)
				diags = diags.Append(blockDiags.UnderPath(path))
			}
		default:
			diags = diags.Append(Diagnostic{
				Severity: Error,
				Summary:  "Unsupported nested block mode",
				Detail:   fmt.Sprintf("Block type %q has an unsupported nested block mode %#v. This is a bug in the provider; please report it in the provider's own issue tracker.", name, blockS.Nesting),
				Path:     path,
			})
		}
	}

	return diags
}

// Validate checks that the given value is a suitable value for the receiving
// attribute, returning diagnostics if not.
//
// This method is usually used only indirectly via SchemaBlockType.Validate.
func (a *SchemaAttribute) Validate(val cty.Value) Diagnostics {
	var diags Diagnostics

	if a.Required && val.IsNull() {
		// This is a poor error message due to our lack of context here. In
		// normal use a whole-schema validation driver should detect this
		// case before calling SchemaAttribute.Validate and return a message
		// with better context.
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Missing required argument",
			Detail:   "This argument is required.",
		})
	}

	convVal, err := convert.Convert(val, a.Type)
	if err != nil {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid argument value",
			Detail:   fmt.Sprintf("Incorrect value type: %s.", err),
		})
	}

	if diags.HasErrors() {
		// If we've already got errors then we'll skip calling the provider's
		// custom validate function, since this avoids the need for that
		// function to be resilient to already-detected problems, and avoids
		// producing duplicate error messages.
		return diags
	}

	if convVal.IsNull() {
		// Null-ness is already handled by the a.Required flag, so if an
		// optional argument is null we'll save the validation function from
		// having to also deal with it.
		return diags
	}

	if !convVal.IsKnown() {
		// If the value isn't known yet then we'll defer any further validation
		// of it until it becomes known, since custom validation functions
		// are not expected to deal with unknown values.
		return diags
	}

	// The validation function gets the already-converted value, for convenience.
	validate, err := wrapSimpleFunction(a.ValidateFn, convVal)
	if err != nil {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid provider schema",
			Detail:   fmt.Sprintf("Invalid ValidateFn: %s.\nThis is a bug in the provider that should be reported in its own issue tracker.", err),
		})
		return diags
	}

	moreDiags := validate()
	diags = diags.Append(moreDiags)
	return diags
}

// ImpliedCtyType derives a cty.Type value to represent values conforming to
// the receiving schema. The returned type is always an object type, with its
// attributes derived from the attributes and nested block types defined in
// the schema.
//
// This corresponds with similar logic in Terraform itself, and so must be
// compatible enough with that logic to communicate with Terraform's own
// object serializer/deserializer.
//
// This function produces reasonable results only for a valid schema. Use
// InternalValidate on the schema in provider tests to check that it is correct.
// When called on an invalid schema, the result may be incorrect or incomplete.
func (b *SchemaBlockType) ImpliedCtyType() cty.Type {
	atys := make(map[string]cty.Type)
	for name, attrS := range b.Attributes {
		atys[name] = attrS.Type
	}
	for name, blockS := range b.NestedBlockTypes {
		atys[name] = blockS.impliedCtyType()
	}
	return cty.Object(atys)
}

func (b *SchemaNestedBlockType) impliedCtyType() cty.Type {
	nested := b.Content.ImpliedCtyType()
	if b.Nesting == SchemaNestingSingle {
		return nested // easy case
	}

	if nested.HasDynamicTypes() {
		// If a multi-nesting block contains any dynamic-typed attributes then
		// it'll be passed in as either a tuple or an object type with full
		// type information in the payload, so for the purposes of our static
		// type constraint, the whole block type attribute is itself
		// dynamically-typed.
		return cty.DynamicPseudoType
	}

	switch b.Nesting {
	case SchemaNestingList:
		return cty.List(nested)
	case SchemaNestingSet:
		return cty.Set(nested)
	case SchemaNestingMap:
		return cty.Map(nested)
	default:
		// Invalid, so what we return here is undefined as far as our godoc is
		// concerned.
		return cty.DynamicPseudoType
	}
}
