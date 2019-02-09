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

	// ValidateFunc, if non-nil, must be set to a function that takes a single
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
	ValidateFunc interface{}
}

type SchemaNestedBlockType struct {
	Nesting SchemaNestingMode
	Content SchemaBlockType
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
func (a *SchemaBlockType) Validate(val cty.Value) Diagnostics {
	var diags Diagnostics
	if !val.Type().IsObjectType() {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid block object",
			Detail:   "An object value is required to represent this block.",
			// TODO: Path
		})
		return diags
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

	if !convVal.IsKnown() {
		// If the value isn't known yet then we'll defer any further validation
		// of it until it becomes known, since custom validation functions
		// are not expected to deal with unknown values.
		return diags
	}

	// The validation function gets the already-converted value, for convenience.
	validate, err := wrapSimpleFunction(a.ValidateFunc, convVal)
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
