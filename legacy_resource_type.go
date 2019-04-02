package tfsdk

import (
	"context"

	"github.com/apparentlymart/terraform-sdk/tflegacy"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/zclconf/go-cty/cty"
)

// LegacyManagedResourceType wraps a managed resource type implemented as
// tflegacy.Resource (formerly schema.Resource) to behave as a
// ManagedResourceType, by applying shims from the new protocol to the older
// types that legacy implementations depend on.
func LegacyManagedResourceType(def *tflegacy.Resource) ManagedResourceType {
	return legacyManagedResourceType{def}
}

// LegacyDataResourceType wraps a data resource type implemented as
// tflegacy.Resource (formerly schema.Resource) to behave as a
// DataResourceType, by applying shims from the new protocol to the older
// types that legacy implementations depend on.
func LegacyDataResourceType(def *tflegacy.Resource) DataResourceType {
	return legacyDataResourceType{def}
}

type legacyManagedResourceType struct {
	r *tflegacy.Resource
}

func (rt legacyManagedResourceType) getSchema() (schema *tfschema.BlockType, version int64) {
	schema = prepareLegacyResourceTypeSchema(rt.r, false)
	version = int64(rt.r.SchemaVersion)
	return
}

func (rt legacyManagedResourceType) validate(obj cty.Value) Diagnostics {
	// TODO: Implement
	panic("not implemented")
}

func (rt legacyManagedResourceType) upgradeState(oldJSON []byte, oldVersion int) (cty.Value, Diagnostics) {
	// TODO: Implement
	panic("not implemented")
}

func (rt legacyManagedResourceType) refresh(ctx context.Context, client interface{}, current cty.Value) (cty.Value, Diagnostics) {
	// TODO: Implement
	panic("not implemented")
}

func (rt legacyManagedResourceType) planChange(ctx context.Context, client interface{}, prior, config, proposed cty.Value) (cty.Value, Diagnostics) {
	// TODO: Implement
	panic("not implemented")
}

func (rt legacyManagedResourceType) applyChange(ctx context.Context, client interface{}, prior, planned cty.Value) (cty.Value, Diagnostics) {
	// TODO: Implement
	panic("not implemented")
}

func (rt legacyManagedResourceType) importState(ctx context.Context, client interface{}, id string) (cty.Value, Diagnostics) {
	// TODO: Implement
	panic("not implemented")
}

type legacyDataResourceType struct {
	r *tflegacy.Resource
}

func (rt legacyDataResourceType) getSchema() *tfschema.BlockType {
	return prepareLegacySchema(rt.r.Schema, false)
}

func (rt legacyDataResourceType) validate(obj cty.Value) Diagnostics {
	// TODO: Implement
	panic("not implemented")
}

func (rt legacyDataResourceType) read(ctx context.Context, client interface{}, config cty.Value) (cty.Value, Diagnostics) {
	// TODO: Implement
	panic("not implemented")
}
