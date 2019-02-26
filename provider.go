package tfsdk

import (
	"context"

	"github.com/zclconf/go-cty/cty"
)

// Provider is the main type for describing a Terraform provider
// implementation. The primary Go package for a provider should include
// a function that returns a pointer to a Provider object describing the
// resource types and other objects exposed by the provider.
type Provider struct {
	ConfigSchema         *SchemaBlockType
	ManagedResourceTypes map[string]ManagedResourceType
	DataResourceTypes    map[string]DataResourceType

	ConfigureFn interface{}
}

// ManagedResourceType is the interface implemented by managed resource type
// implementations.
//
// This is a closed interface, meaning that all of its implementations are
// inside this package. To implement a managed resource type, create a
// *ResourceType value and pass it to NewManagedResourceType.
type ManagedResourceType interface {
	getSchema() (schema *SchemaBlockType, version int64)
	validate(obj cty.Value) Diagnostics
	upgradeState(oldJSON []byte, oldVersion int) (cty.Value, Diagnostics)
	refresh(ctx context.Context, client interface{}, old cty.Value) (cty.Value, Diagnostics)
	planChange(ctx context.Context, client interface{}, prior, config, proposed cty.Value) (cty.Value, Diagnostics)
	applyChange(ctx context.Context, client interface{}, prior, config, planned cty.Value) (cty.Value, Diagnostics)
	importState(ctx context.Context, client interface{}, id string) (cty.Value, Diagnostics)
}

// DataResourceType is an interface implemented by data resource type
// implementations.
//
// This is a closed interface, meaning that all of its implementations are
// inside this package. To implement a managed resource type, create a
// *ResourceType value and pass it to NewDataResourceType.
type DataResourceType interface {
	getSchema() *SchemaBlockType
	validate(obj cty.Value) Diagnostics
	read(ctx context.Context, client interface{}, config cty.Value) (cty.Value, Diagnostics)
}

// PrepareConfig accepts an object decoded from the user-provided configuration
// (whose type must conform to the schema) and validates it, possibly also
// altering some of the values within to produce a final configuration for
// Terraform Core to use when interacting with this provider instance.
func (p *Provider) PrepareConfig(proposedVal cty.Value) (cty.Value, Diagnostics) {
	diags := p.ConfigSchema.Validate(proposedVal)
	return proposedVal, diags
}
