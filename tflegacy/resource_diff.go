package tflegacy

// ResourceDiff is used to query and make custom changes to an in-flight diff.
// It can be used to veto particular changes in the diff, customize the diff
// that has been created, or diff values not controlled by config.
//
// The object functions similar to ResourceData, however most notably lacks
// Set, SetPartial, and Partial, as it should be used to change diff values
// only.  Most other first-class ResourceData functions exist, namely Get,
// GetOk, HasChange, and GetChange exist.
//
// All functions in ResourceDiff, save for ForceNew, can only be used on
// computed fields.
type ResourceDiff struct {
	// The schema for the resource being worked on.
	schema map[string]*Schema

	// The current config for this resource.
	config *ResourceConfig

	// The state for this resource as it exists post-refresh, after the initial
	// diff.
	state *InstanceState

	// The diff created by Terraform. This diff is used, along with state,
	// config, and custom-set diff data, to provide a multi-level reader
	// experience similar to ResourceData.
	diff *InstanceDiff

	// The internal reader structure that contains the state, config, the default
	// diff, and the new diff.
	//multiReader *MultiLevelFieldReader

	// A writer that writes overridden new fields.
	//newWriter *newValueWriter

	// Tracks which keys have been updated by ResourceDiff to ensure that the
	// diff does not get re-run on keys that were not touched, or diffs that were
	// just removed (re-running on the latter would just roll back the removal).
	updatedKeys map[string]bool

	// Tracks which keys were flagged as forceNew. These keys are not saved in
	// newWriter, but we need to track them so that they can be re-diffed later.
	forcedNewKeys map[string]bool
}
