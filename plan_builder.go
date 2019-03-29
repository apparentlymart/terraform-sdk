package tfsdk

import (
	"github.com/zclconf/go-cty/cty"
)

// PlanBuilder is an extension of ObjectBuilder that provides access to
// information about the prior state and configuration that a plan is being
// built for.
//
// The object being built by a PlanBuilder is the "planned new object" for
// a change to a managed resource instance, and so as such it must reflect
// the provider's best possible prediction of what new object will result for
// the resource instance after the plan is applied. Unknown values must be
// used as placeholders for any attribute values the provider cannot predict
// during the plan phase; any known attribute values in the planned object are
// required to exactly match the final result, or Terraform Core will consider
// that a bug in the provider and abort the apply operation.
type PlanBuilder interface {
	ObjectBuilder

	// PriorReader returns an ObjectReader for the prior object when planning
	// for an update operation. Returns nil when planning for a create
	// operation, because there is no prior object in that case.
	PriorReader() ObjectReader

	// ConfigReader returns an ObjectReader for the object representing the
	// configuration as written by the user. The config object has values
	// only for attributes that were set in the configuration; all other
	// attributes have null values, allowing the provider to determine whether
	// it is appropriate to substitute a default value for an attribute that
	// is marked as Computed.
	ConfigReader() ObjectReader

	// AttrChange returns the value of the given attribute from the prior
	// object and the planned new object respectively. When planning for
	// a "create" operation, the prior object is always null.
	AttrChange(name string) (prior, planned cty.Value)

	// AttrHasChange returns true if the prior value for the attribute of the
	// given name is different than the planned new value for that same
	// attribute.
	AttrHasChange(name string) bool

	// CanProvideAttrDefault returns true if and only if the attribute of the
	// given name is marked as Computed in the schema and that attribute has
	// a null value in the user configuration. In that case, a provider is
	// permitted to provide a default value during the plan phase, which might
	// be an unknown value if the final result will not be known until the
	// apply phase.
	//
	// PlanBuilder won't prevent attempts to set defaults that violate these
	// rules, but Terraform Core itself will reject any plan that contradicts
	// explicit values given by the user in configuration.
	CanProvideAttrDefault(name string) bool

	// SetAttrUnknown is equivalent to calling SetAttr with an unknown value
	// of the appropriate type for the given attribute. It just avoids the
	// need for the caller to construct such a value.
	SetAttrUnknown(name string)

	// SetAttrNull is equivalent to calling SetAttr with a null value
	// of the appropriate type for the given attribute. It just avoids the
	// need for the caller to construct such a value.
	SetAttrNull(name string)

	// The BlockPlanBuilder... family of methods echoes the BlockBuilder...
	// family of methods from the ObjectBuilder type but they each return
	// a PlanBuilder for the corresponding requested block(s), rather than just
	// an ObjectBuilder.
	//
	// A plan is not permitted to change the collection of blocks, only to
	// provide information about the results of nested attributes that are
	// marked as Computed in the schema nad that have not been set in
	// configuration.
	BlockPlanBuilderSingle(blockType string) ObjectBuilder
	BlockPlanBuilderList(blockType string) []ObjectBuilder
	BlockPlanBuilderMap(blockType string) map[string]ObjectBuilder
	BlockPlanBuilderFromList(blockType string, idx int) ObjectBuilder
	BlockPlanBuilderFromMap(blockType string, key string) ObjectBuilder
}
