package tfsdk

import (
	"context"
	"log"

	"github.com/zclconf/go-cty/cty"
)

// ResourceType is the type that provider packages should instantiate to
// implement a specific resource type.
//
// Pointers to instances of this type can be passed to the functions
// NewManagedResourceType and NewDataResourceType to provide managed and
// data resource type implementations respectively.
type ResourceType struct {
	ConfigSchema  *SchemaBlockType
	SchemaVersion int64 // Only used for managed resource types; leave as zero otherwise
}

// NewManagedResourceType prepares a ManagedResourceType implementation using
// the definition from the given ResourceType instance.
//
// This function is intended to be called during startup with a valid
// ResourceType, so it will panic if the given ResourceType is not valid.
func NewManagedResourceType(def *ResourceType) ManagedResourceType {
	if def == nil {
		panic("NewManagedResourceType called with nil definition")
	}

	schema := def.ConfigSchema
	if schema == nil {
		schema = &SchemaBlockType{}
	}

	// TODO: Check thoroughly to make sure def is correctly populated for a
	// managed resource type, so we can panic early.

	return managedResourceType{
		configSchema: schema,
	}
}

// NewDataResourceType prepares a DataResourceType implementation using the
// definition from the given ResourceType instance.
//
// This function is intended to be called during startup with a valid
// ResourceType, so it will panic if the given ResourceType is not valid.
func NewDataResourceType(def *ResourceType) DataResourceType {
	if def == nil {
		panic("NewDataResourceType called with nil definition")
	}

	schema := def.ConfigSchema
	if schema == nil {
		schema = &SchemaBlockType{}
	}
	if def.SchemaVersion != 0 {
		panic("NewDataResourceType requires def.SchemaVersion == 0")
	}

	// TODO: Check thoroughly to make sure def is correctly populated for a data
	// resource type, so we can panic early.
	return dataResourceType{
		configSchema: schema,
	}
}

type managedResourceType struct {
	configSchema  *SchemaBlockType
	schemaVersion int64
}

func (rt managedResourceType) getSchema() (schema *SchemaBlockType, version int64) {
	return rt.configSchema, rt.schemaVersion
}

func (rt managedResourceType) validate(obj cty.Value) Diagnostics {
	return rt.configSchema.Validate(obj)
}

func (rt managedResourceType) upgradeState(oldJSON []byte, oldVersion int) (cty.Value, Diagnostics) {
	return cty.NilVal, nil
}

func (rt managedResourceType) refresh(ctx context.Context, client interface{}, old cty.Value) (cty.Value, Diagnostics) {
	return cty.NilVal, nil
}

func (rt managedResourceType) planChange(ctx context.Context, client interface{}, prior, config, proposed cty.Value) (cty.Value, Diagnostics) {
	var diags Diagnostics

	// Terraform Core has already done a lot of the work in merging prior with
	// config to produce "proposed". Our main job here is inserting any additional
	// default values called for in the provider schema.
	planned := rt.configSchema.ApplyDefaults(proposed)
	log.Printf("applied defaults\n    before: %#v\n    after:  %#v", proposed, planned)

	// TODO: We should also give the provider code an opportunity to make
	// further changes to the "Computed" parts of the planned value so it
	// can use its own logic, or possibly remote API calls, to produce the
	// most accurate plan.

	return planned, diags
}

func (rt managedResourceType) applyChange(ctx context.Context, client interface{}, prior, config, planned cty.Value) (cty.Value, Diagnostics) {
	return cty.NilVal, nil
}

func (rt managedResourceType) importState(ctx context.Context, client interface{}, id string) (cty.Value, Diagnostics) {
	return cty.NilVal, nil
}

type dataResourceType struct {
	configSchema *SchemaBlockType
}

func (rt dataResourceType) getSchema() *SchemaBlockType {
	return rt.configSchema
}

func (rt dataResourceType) validate(obj cty.Value) Diagnostics {
	return rt.configSchema.Validate(obj)
}

func (rt dataResourceType) read(ctx context.Context, client interface{}, config cty.Value) (cty.Value, Diagnostics) {
	return cty.NilVal, nil
}
