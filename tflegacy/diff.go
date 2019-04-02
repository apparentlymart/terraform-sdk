package tflegacy

import (
	"fmt"
	"sync"
)

// InstanceDiff is the diff of a resource from some state to another.
type InstanceDiff struct {
	mu             sync.Mutex
	Attributes     map[string]*ResourceAttrDiff
	Destroy        bool
	DestroyDeposed bool
	DestroyTainted bool

	// Meta is a simple K/V map that is stored in a diff and persisted to
	// plans but otherwise is completely ignored by Terraform core. It is
	// meant to be used for additional data a resource may want to pass through.
	// The value here must only contain Go primitives and collections.
	Meta map[string]interface{}
}

func (d *InstanceDiff) Lock()   { d.mu.Lock() }
func (d *InstanceDiff) Unlock() { d.mu.Unlock() }

// ResourceAttrDiff is the diff of a single attribute of a resource.
type ResourceAttrDiff struct {
	Old         string      // Old Value
	New         string      // New Value
	NewComputed bool        // True if new value is computed (unknown currently)
	NewRemoved  bool        // True if this attribute is being removed
	NewExtra    interface{} // Extra information for the provider
	RequiresNew bool        // True if change requires new resource
	Sensitive   bool        // True if the data should not be displayed in UI output
	Type        DiffAttrType
}

// Empty returns true if the diff for this attr is neutral
func (d *ResourceAttrDiff) Empty() bool {
	return d.Old == d.New && !d.NewComputed && !d.NewRemoved
}

func (d *ResourceAttrDiff) GoString() string {
	return fmt.Sprintf("*%#v", *d)
}

// DiffAttrType is an enum type that says whether a resource attribute
// diff is an input attribute (comes from the configuration) or an
// output attribute (comes as a result of applying the configuration). An
// example input would be "ami" for AWS and an example output would be
// "private_ip".
type DiffAttrType byte

const (
	DiffAttrUnknown DiffAttrType = iota
	DiffAttrInput
	DiffAttrOutput
)
