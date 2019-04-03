package tflegacy

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/copystructure"
	"github.com/mitchellh/reflectwalk"

	"github.com/apparentlymart/terraform-sdk/internal/shim"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/zclconf/go-cty/cty"
)

type ResourceMode int

//go:generate stringer -type=ResourceMode -output=resource_mode_string.go config.go

const (
	ManagedResourceMode ResourceMode = iota
	DataResourceMode
)

// ResourceConfig is a legacy type that was formerly used to represent
// interpolatable configuration blocks. It is now only used to shim to old
// APIs that still use this type, via NewResourceConfigShimmed.
type ResourceConfig struct {
	ComputedKeys []string
	Raw          map[string]interface{}
	Config       map[string]interface{}
}

// NewResourceConfigShimmed constructs a new ResourceConfig by converting the
// given object value (which must conform to the given schema) into the legacy
// configuration representation.
func NewResourceConfigShimmed(val cty.Value, schema *tfschema.BlockType) *ResourceConfig {
	if !val.Type().IsObjectType() {
		panic(fmt.Sprintf("NewResourceConfigShimmed given %#v; an object type is required", val.Type()))
	}
	ret := &ResourceConfig{}

	legacyVal := shim.ConfigValueFromHCL2Block(val, schema)
	if legacyVal != nil {
		ret.Config = legacyVal

		// Now we need to walk through our structure and find any unknown values,
		// producing the separate list ComputedKeys to represent these. We use the
		// schema here so that we can preserve the expected invariant
		// that an attribute is always either wholly known or wholly unknown, while
		// a child block can be partially unknown.
		ret.ComputedKeys = newResourceConfigShimmedComputedKeys(val, schema, "")
	} else {
		ret.Config = make(map[string]interface{})
	}
	ret.Raw = ret.Config

	return ret
}

// newResourceConfigShimmedComputedKeys finds all of the unknown values in the
// given object, which must conform to the given schema, returning them in
// the format that's expected for ResourceConfig.ComputedKeys.
func newResourceConfigShimmedComputedKeys(obj cty.Value, schema *tfschema.BlockType, prefix string) []string {
	var ret []string
	ty := obj.Type()

	if schema == nil {
		return nil
	}

	for attrName := range schema.Attributes {
		if !ty.HasAttribute(attrName) {
			// Should never happen, but we'll tolerate it anyway
			continue
		}

		attrVal := obj.GetAttr(attrName)
		if !attrVal.IsWhollyKnown() {
			ret = append(ret, prefix+attrName)
		}
	}

	for typeName, blockS := range schema.NestedBlockTypes {
		if !ty.HasAttribute(typeName) {
			// Should never happen, but we'll tolerate it anyway
			continue
		}

		blockVal := obj.GetAttr(typeName)
		if blockVal.IsNull() || !blockVal.IsKnown() {
			continue
		}

		switch blockS.Nesting {
		case tfschema.NestingSingle:
			keys := newResourceConfigShimmedComputedKeys(blockVal, &blockS.Content, fmt.Sprintf("%s%s.", prefix, typeName))
			ret = append(ret, keys...)
		case tfschema.NestingList, tfschema.NestingSet:
			// Producing computed keys items for sets is not really useful
			// since they are not usefully addressable anyway, but we'll treat
			// them like lists just so that ret.ComputedKeys accounts for them
			// all. Our legacy system didn't support sets here anyway, so
			// treating them as lists is the most accurate translation. Although
			// set traversal isn't in any particular order, it is _stable_ as
			// long as the list isn't mutated, and so we know we'll see the
			// same order here as hcl2shim.ConfigValueFromHCL2 would've seen
			// inside NewResourceConfigShimmed above.
			i := 0
			for it := blockVal.ElementIterator(); it.Next(); i++ {
				_, subVal := it.Element()
				subPrefix := fmt.Sprintf("%s.%s%d.", typeName, prefix, i)
				keys := newResourceConfigShimmedComputedKeys(subVal, &blockS.Content, subPrefix)
				ret = append(ret, keys...)
			}
		case tfschema.NestingMap:
			for it := blockVal.ElementIterator(); it.Next(); {
				subK, subVal := it.Element()
				subPrefix := fmt.Sprintf("%s.%s%s.", typeName, prefix, subK.AsString())
				keys := newResourceConfigShimmedComputedKeys(subVal, &blockS.Content, subPrefix)
				ret = append(ret, keys...)
			}
		default:
			// Should never happen, since the above is exhaustive.
			panic(fmt.Errorf("unsupported block nesting type %s", blockS.Nesting))
		}
	}

	return ret
}

// DeepCopy performs a deep copy of the configuration. This makes it safe
// to modify any of the structures that are part of the resource config without
// affecting the original configuration.
func (c *ResourceConfig) DeepCopy() *ResourceConfig {
	// DeepCopying a nil should return a nil to avoid panics
	if c == nil {
		return nil
	}

	// Copy, this will copy all the exported attributes
	copy, err := copystructure.Config{Lock: true}.Copy(c)
	if err != nil {
		panic(err)
	}

	// Force the type
	result := copy.(*ResourceConfig)

	return result
}

// Equal checks the equality of two resource configs.
func (c *ResourceConfig) Equal(c2 *ResourceConfig) bool {
	// If either are nil, then they're only equal if they're both nil
	if c == nil || c2 == nil {
		return c == c2
	}

	// Sort the computed keys so they're deterministic
	sort.Strings(c.ComputedKeys)
	sort.Strings(c2.ComputedKeys)

	// Two resource configs if their exported properties are equal.
	// We don't compare "raw" because it is never used again after
	// initialization and for all intents and purposes they are equal
	// if the exported properties are equal.
	check := [][2]interface{}{
		{c.ComputedKeys, c2.ComputedKeys},
		{c.Raw, c2.Raw},
		{c.Config, c2.Config},
	}
	for _, pair := range check {
		if !reflect.DeepEqual(pair[0], pair[1]) {
			return false
		}
	}

	return true
}

// CheckSet checks that the given list of configuration keys is
// properly set. If not, errors are returned for each unset key.
//
// This is useful to be called in the Validate method of a ResourceProvider.
func (c *ResourceConfig) CheckSet(keys []string) []error {
	var errs []error

	for _, k := range keys {
		if !c.IsSet(k) {
			errs = append(errs, fmt.Errorf("%s must be set", k))
		}
	}

	return errs
}

// Get looks up a configuration value by key and returns the value.
//
// The second return value is true if the get was successful. Get will
// return the raw value if the key is computed, so you should pair this
// with IsComputed.
func (c *ResourceConfig) Get(k string) (interface{}, bool) {
	// We aim to get a value from the configuration. If it is computed,
	// then we return the pure raw value.
	source := c.Config
	if c.IsComputed(k) {
		source = c.Raw
	}

	return c.get(k, source)
}

// GetRaw looks up a configuration value by key and returns the value,
// from the raw, uninterpolated config.
//
// The second return value is true if the get was successful. Get will
// not succeed if the value is being computed.
func (c *ResourceConfig) GetRaw(k string) (interface{}, bool) {
	return c.get(k, c.Raw)
}

// IsComputed returns whether the given key is computed or not.
func (c *ResourceConfig) IsComputed(k string) bool {
	// The next thing we do is check the config if we get a computed
	// value out of it.
	v, ok := c.get(k, c.Config)
	if !ok {
		return false
	}

	// If value is nil, then it isn't computed
	if v == nil {
		return false
	}

	// Test if the value contains an unknown value
	var w unknownCheckWalker
	if err := reflectwalk.Walk(v, &w); err != nil {
		panic(err)
	}

	return w.Unknown
}

// IsSet checks if the key in the configuration is set. A key is set if
// it has a value or the value is being computed (is unknown currently).
//
// This function should be used rather than checking the keys of the
// raw configuration itself, since a key may be omitted from the raw
// configuration if it is being computed.
func (c *ResourceConfig) IsSet(k string) bool {
	if c == nil {
		return false
	}

	if c.IsComputed(k) {
		return true
	}

	if _, ok := c.Get(k); ok {
		return true
	}

	return false
}

func (c *ResourceConfig) get(
	k string, raw map[string]interface{}) (interface{}, bool) {
	parts := strings.Split(k, ".")
	if len(parts) == 1 && parts[0] == "" {
		parts = nil
	}

	var current interface{} = raw
	var previous interface{} = nil
	for i, part := range parts {
		if current == nil {
			return nil, false
		}

		cv := reflect.ValueOf(current)
		switch cv.Kind() {
		case reflect.Map:
			previous = current
			v := cv.MapIndex(reflect.ValueOf(part))
			if !v.IsValid() {
				if i > 0 && i != (len(parts)-1) {
					tryKey := strings.Join(parts[i:], ".")
					v := cv.MapIndex(reflect.ValueOf(tryKey))
					if !v.IsValid() {
						return nil, false
					}

					return v.Interface(), true
				}

				return nil, false
			}

			current = v.Interface()
		case reflect.Slice:
			previous = current

			if part == "#" {
				// If any value in a list is computed, this whole thing
				// is computed and we can't read any part of it.
				for i := 0; i < cv.Len(); i++ {
					if v := cv.Index(i).Interface(); v == shim.UnknownVariableValue {
						return v, true
					}
				}

				current = cv.Len()
			} else {
				i, err := strconv.ParseInt(part, 0, 0)
				if err != nil {
					return nil, false
				}
				if int(i) < 0 || int(i) >= cv.Len() {
					return nil, false
				}
				current = cv.Index(int(i)).Interface()
			}
		case reflect.String:
			// This happens when map keys contain "." and have a common
			// prefix so were split as path components above.
			actualKey := strings.Join(parts[i-1:], ".")
			if prevMap, ok := previous.(map[string]interface{}); ok {
				v, ok := prevMap[actualKey]
				return v, ok
			}

			return nil, false
		default:
			panic(fmt.Sprintf("Unknown kind: %s", cv.Kind()))
		}
	}

	return current, true
}

// unknownCheckWalker
type unknownCheckWalker struct {
	Unknown bool
}

func (w *unknownCheckWalker) Primitive(v reflect.Value) error {
	if v.Interface() == shim.UnknownVariableValue {
		w.Unknown = true
	}

	return nil
}
