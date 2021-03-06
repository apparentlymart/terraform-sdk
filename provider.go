package tfsdk

import (
	"context"
	"fmt"

	"github.com/apparentlymart/terraform-sdk/internal/dynfunc"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/zclconf/go-cty/cty"
)

// Provider is the main type for describing a Terraform provider
// implementation. The primary Go package for a provider should include
// a function that returns a pointer to a Provider object describing the
// resource types and other objects exposed by the provider.
type Provider struct {
	ConfigSchema         *tfschema.BlockType
	ManagedResourceTypes map[string]ManagedResourceType
	DataResourceTypes    map[string]DataResourceType

	ConfigureFn interface{}

	client interface{}
}

// ManagedResourceType is the interface implemented by managed resource type
// implementations.
//
// This is a closed interface, meaning that all of its implementations are
// inside this package. To implement a managed resource type, create a
// *ResourceType value and pass it to NewManagedResourceType.
type ManagedResourceType interface {
	getSchema() (schema *tfschema.BlockType, version int64)
	validate(obj cty.Value) Diagnostics
	upgradeState(oldJSON []byte, oldVersion int) (cty.Value, Diagnostics)
	refresh(ctx context.Context, client interface{}, old cty.Value) (cty.Value, Diagnostics)
	planChange(ctx context.Context, client interface{}, prior, config, proposed cty.Value) (planned cty.Value, requiresReplace cty.PathSet, diags Diagnostics)
	applyChange(ctx context.Context, client interface{}, prior, planned cty.Value) (cty.Value, Diagnostics)
	importState(ctx context.Context, client interface{}, id string) (cty.Value, Diagnostics)
}

// DataResourceType is an interface implemented by data resource type
// implementations.
//
// This is a closed interface, meaning that all of its implementations are
// inside this package. To implement a managed resource type, create a
// *ResourceType value and pass it to NewDataResourceType.
type DataResourceType interface {
	getSchema() *tfschema.BlockType
	validate(obj cty.Value) Diagnostics
	read(ctx context.Context, client interface{}, config cty.Value) (cty.Value, Diagnostics)
}

// prepareConfig accepts an object decoded from the user-provided configuration
// (whose type must conform to the schema) and validates it, possibly also
// altering some of the values within to produce a final configuration for
// Terraform Core to use when interacting with this provider instance.
func (p *Provider) prepareConfig(proposedVal cty.Value) (cty.Value, Diagnostics) {
	diags := ValidateBlockObject(p.ConfigSchema, proposedVal)
	return proposedVal, diags
}

// configure recieves the finalized configuration for the provider and passes
// it to the provider's configuration function to produce the client object
// that will be recieved by the various resource operations.
func (p *Provider) configure(ctx context.Context, config cty.Value) Diagnostics {
	var diags Diagnostics
	var client interface{}
	fn, err := dynfunc.WrapFunctionWithReturnValue(p.ConfigureFn, &client, ctx, config)
	if err != nil {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid provider implementation",
			Detail:   fmt.Sprintf("Invalid ConfigureFn: %s.\nThis is a bug in the provider that should be reported in its own issue tracker.", err),
		})
		return diags
	}

	moreDiags := fn()
	diags = diags.Append(moreDiags)
	if !diags.HasErrors() {
		p.client = client
	}
	return diags
}

func (p *Provider) managedResourceType(typeName string) ManagedResourceType {
	return p.ManagedResourceTypes[typeName]
}

func (p *Provider) dataResourceType(typeName string) DataResourceType {
	return p.DataResourceTypes[typeName]
}

func (p *Provider) readResource(ctx context.Context, rt ManagedResourceType, currentVal cty.Value) (cty.Value, Diagnostics) {
	return rt.refresh(ctx, p.client, currentVal)
}

func (p *Provider) readDataSource(ctx context.Context, rt DataResourceType, configVal cty.Value) (cty.Value, Diagnostics) {
	return rt.read(ctx, p.client, configVal)
}

func (p *Provider) planResourceChange(ctx context.Context, rt ManagedResourceType, priorVal, configVal, proposedVal cty.Value) (cty.Value, cty.PathSet, Diagnostics) {
	return rt.planChange(ctx, p.client, priorVal, configVal, proposedVal)
}

func (p *Provider) applyResourceChange(ctx context.Context, rt ManagedResourceType, priorVal, plannedVal cty.Value) (cty.Value, Diagnostics) {
	return rt.applyChange(ctx, p.client, priorVal, plannedVal)
}
