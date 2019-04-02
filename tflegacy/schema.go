package tflegacy

import (
	"fmt"
	"os"
)

//go:generate stringer -type=ValueType

// ValueType is an enum of the type kinds that can be represented by a schema.
type ValueType int

const (
	TypeInvalid ValueType = iota
	TypeBool
	TypeInt
	TypeFloat
	TypeString
	TypeList
	TypeMap
	TypeSet
	typeObject
)

// Schema is used by older providers to describe a single attribute or a nested
// block type, depending on the value of ConfigMode.
//
// Read the documentation of the struct fields for important details.
type Schema struct {
	// Type is the type of the value and must be one of the ValueType values.
	//
	// This type not only determines what type is expected/valid in configuring
	// this value, but also what type is returned when ResourceData.Get is
	// called. The types returned by Get are:
	//
	//   TypeBool - bool
	//   TypeInt - int
	//   TypeFloat - float64
	//   TypeString - string
	//   TypeList - []interface{}
	//   TypeMap - map[string]interface{}
	//   TypeSet - *schema.Set
	//
	Type ValueType

	// ConfigMode allows for overriding the default behaviors for mapping
	// schema entries onto configuration constructs.
	//
	// By default, the Elem field is used to choose whether a particular
	// schema is represented in configuration as an attribute or as a nested
	// block; if Elem is a *schema.Resource then it's a block and it's an
	// attribute otherwise.
	//
	// If Elem is *schema.Resource then setting ConfigMode to
	// SchemaConfigModeAttr will force it to be represented in configuration
	// as an attribute, which means that the Computed flag can be used to
	// provide default elements when the argument isn't set at all, while still
	// allowing the user to force zero elements by explicitly assigning an
	// empty list.
	//
	// When Computed is set without Optional, the attribute is not settable
	// in configuration at all and so SchemaConfigModeAttr is the automatic
	// behavior, and SchemaConfigModeBlock is not permitted.
	ConfigMode SchemaConfigMode

	// If one of these is set, then this item can come from the configuration.
	// Both cannot be set. If Optional is set, the value is optional. If
	// Required is set, the value is required.
	//
	// One of these must be set if the value is not computed. That is:
	// value either comes from the config, is computed, or is both.
	Optional bool
	Required bool

	// If this is non-nil, the provided function will be used during diff
	// of this field. If this is nil, a default diff for the type of the
	// schema will be used.
	//
	// This allows comparison based on something other than primitive, list
	// or map equality - for example SSH public keys may be considered
	// equivalent regardless of trailing whitespace.
	DiffSuppressFunc SchemaDiffSuppressFunc

	// If this is non-nil, then this will be a default value that is used
	// when this item is not set in the configuration.
	//
	// DefaultFunc can be specified to compute a dynamic default.
	// Only one of Default or DefaultFunc can be set. If DefaultFunc is
	// used then its return value should be stable to avoid generating
	// confusing/perpetual diffs.
	//
	// Changing either Default or the return value of DefaultFunc can be
	// a breaking change, especially if the attribute in question has
	// ForceNew set. If a default needs to change to align with changing
	// assumptions in an upstream API then it may be necessary to also use
	// the MigrateState function on the resource to change the state to match,
	// or have the Read function adjust the state value to align with the
	// new default.
	//
	// If Required is true above, then Default cannot be set. DefaultFunc
	// can be set with Required. If the DefaultFunc returns nil, then there
	// will be no default and the user will be asked to fill it in.
	//
	// If either of these is set, then the user won't be asked for input
	// for this key if the default is not nil.
	Default     interface{}
	DefaultFunc SchemaDefaultFunc

	// Description is used as the description for docs or asking for user
	// input. It should be relatively short (a few sentences max) and should
	// be formatted to fit a CLI.
	Description string

	// InputDefault is the default value to use for when inputs are requested.
	// This differs from Default in that if Default is set, no input is
	// asked for. If Input is asked, this will be the default value offered.
	InputDefault string

	// The fields below relate to diffs.
	//
	// If Computed is true, then the result of this value is computed
	// (unless specified by config) on creation.
	//
	// If ForceNew is true, then a change in this resource necessitates
	// the creation of a new resource.
	//
	// StateFunc is a function called to change the value of this before
	// storing it in the state (and likewise before comparing for diffs).
	// The use for this is for example with large strings, you may want
	// to simply store the hash of it.
	Computed  bool
	ForceNew  bool
	StateFunc SchemaStateFunc

	// The following fields are only set for a TypeList, TypeSet, or TypeMap.
	//
	// Elem represents the element type. For a TypeMap, it must be a *Schema
	// with a Type of TypeString, otherwise it may be either a *Schema or a
	// *Resource. If it is *Schema, the element type is just a simple value.
	// If it is *Resource, the element type is a complex structure,
	// potentially with its own lifecycle.
	Elem interface{}

	// The following fields are only set for a TypeList or TypeSet.
	//
	// MaxItems defines a maximum amount of items that can exist within a
	// TypeSet or TypeList. Specific use cases would be if a TypeSet is being
	// used to wrap a complex structure, however more than one instance would
	// cause instability.
	//
	// MinItems defines a minimum amount of items that can exist within a
	// TypeSet or TypeList. Specific use cases would be if a TypeSet is being
	// used to wrap a complex structure, however less than one instance would
	// cause instability.
	//
	// If the field Optional is set to true then MinItems is ignored and thus
	// effectively zero.
	//
	// If MaxItems is 1, you may optionally also set AsSingle in order to have
	// Terraform v0.12 or later treat a TypeList or TypeSet as if it were a
	// single value. It will remain a list or set in Terraform v0.10 and v0.11.
	// Enabling this for an existing attribute after you've made at least one
	// v0.12-compatible provider release is a breaking change. AsSingle is
	// likely to misbehave when used with deeply-nested set structures due to
	// the imprecision of set diffs, so be sure to test it thoroughly,
	// including updates that change the set members at all levels. AsSingle
	// exists primarily to be used in conjunction with ConfigMode when forcing
	// a nested resource to be treated as an attribute, so it can be considered
	// an attribute of object type rather than of list/set of object.
	MaxItems int
	MinItems int
	AsSingle bool

	// PromoteSingle originally allowed for a single element to be assigned
	// where a primitive list was expected, but this no longer works from
	// Terraform v0.12 onwards (Terraform Core will require a list to be set
	// regardless of what this is set to) and so only applies to Terraform v0.11
	// and earlier, and so should be used only to retain this functionality
	// for those still using v0.11 with a provider that formerly used this.
	PromoteSingle bool

	// The following fields are only valid for a TypeSet type.
	//
	// Set defines a function to determine the unique ID of an item so that
	// a proper set can be built.
	Set SchemaSetFunc

	// ComputedWhen is a set of queries on the configuration. Whenever any
	// of these things is changed, it will require a recompute (this requires
	// that Computed is set to true).
	//
	// NOTE: This currently does not work.
	ComputedWhen []string

	// ConflictsWith is a set of schema keys that conflict with this schema.
	// This will only check that they're set in the _config_. This will not
	// raise an error for a malfunctioning resource that sets a conflicting
	// key.
	ConflictsWith []string

	// When Deprecated is set, this attribute is deprecated.
	//
	// A deprecated field still works, but will probably stop working in near
	// future. This string is the message shown to the user with instructions on
	// how to address the deprecation.
	Deprecated string

	// When Removed is set, this attribute has been removed from the schema
	//
	// Removed attributes can be left in the Schema to generate informative error
	// messages for the user when they show up in resource configurations.
	// This string is the message shown to the user with instructions on
	// what do to about the removed attribute.
	Removed string

	// ValidateFunc allows individual fields to define arbitrary validation
	// logic. It is yielded the provided config value as an interface{} that is
	// guaranteed to be of the proper Schema type, and it can yield warnings or
	// errors based on inspection of that value.
	//
	// ValidateFunc currently only works for primitive types.
	ValidateFunc SchemaValidateFunc

	// Sensitive ensures that the attribute's value does not get displayed in
	// logs or regular output. It should be used for passwords or other
	// secret fields. Future versions of Terraform may encrypt these
	// values.
	Sensitive bool
}

// SchemaConfigMode is used to influence how a schema item is mapped into a
// corresponding configuration construct, using the ConfigMode field of
// Schema.
type SchemaConfigMode int

const (
	SchemaConfigModeAuto SchemaConfigMode = iota
	SchemaConfigModeAttr
	SchemaConfigModeBlock
)

// SchemaDiffSuppressFunc is a function which can be used to determine
// whether a detected diff on a schema element is "valid" or not, and
// suppress it from the plan if necessary.
//
// Return true if the diff should be suppressed, false to retain it.
//type SchemaDiffSuppressFunc func(k, old, new string, d *ResourceData) bool

// SchemaDefaultFunc is a function called to return a default value for
// a field.
type SchemaDefaultFunc func() (interface{}, error)

// SchemaDiffSuppressFunc is a function which can be used to determine
// whether a detected diff on a schema element is "valid" or not, and
// suppress it from the plan if necessary.
//
// Return true if the diff should be suppressed, false to retain it.
type SchemaDiffSuppressFunc func(k, old, new string, d *ResourceData) bool

// EnvDefaultFunc is a helper function that returns the value of the
// given environment variable, if one exists, or the default value
// otherwise.
func EnvDefaultFunc(k string, dv interface{}) SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return dv, nil
	}
}

// MultiEnvDefaultFunc is a helper function that returns the value of the first
// environment variable in the given list that returns a non-empty value. If
// none of the environment variables return a value, the default value is
// returned.
func MultiEnvDefaultFunc(ks []string, dv interface{}) SchemaDefaultFunc {
	return func() (interface{}, error) {
		for _, k := range ks {
			if v := os.Getenv(k); v != "" {
				return v, nil
			}
		}
		return dv, nil
	}
}

// SchemaSetFunc is a function that must return a unique ID for the given
// element. This unique ID is used to store the element in a hash.
type SchemaSetFunc func(interface{}) int

// SchemaStateFunc is a function used to convert some type to a string
// to be stored in the state.
type SchemaStateFunc func(interface{}) string

// SchemaValidateFunc is a function used to validate a single field in the
// schema.
type SchemaValidateFunc func(interface{}, string) ([]string, []error)

func (s *Schema) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}
