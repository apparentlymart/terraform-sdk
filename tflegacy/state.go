package tflegacy

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// State is a snapshot of the state in a legacy format.
type State struct {
	// Version is the state file protocol version.
	Version int `json:"version"`

	// TFVersion is the version of Terraform that wrote this state.
	TFVersion string `json:"terraform_version,omitempty"`

	// Serial is incremented on any operation that modifies
	// the State file. It is used to detect potentially conflicting
	// updates.
	Serial int64 `json:"serial"`

	// Lineage is set when a new, blank state is created and then
	// never updated. This allows us to determine whether the serials
	// of two states can be meaningfully compared.
	// Apart from the guarantee that collisions between two lineages
	// are very unlikely, this value is opaque and external callers
	// should only compare lineage strings byte-for-byte for equality.
	Lineage string `json:"lineage"`

	// Modules contains all the modules in a breadth-first order
	Modules []*ModuleState `json:"modules"`

	mu sync.Mutex
}

// OutputState is a legacy representation of an output value from a State.
type OutputState struct {
	// Sensitive describes whether the output is considered sensitive,
	// which may lead to masking the value on screen in some cases.
	Sensitive bool `json:"sensitive"`
	// Type describes the structure of Value. Valid values are "string",
	// "map" and "list"
	Type string `json:"type"`
	// Value contains the value of the output, in the structure described
	// by the Type field.
	Value interface{} `json:"value"`

	mu sync.Mutex
}

// ModuleState is a legacy representation of a module from a State.
type ModuleState struct {
	// Path is the import path from the root module. Modules imports are
	// always disjoint, so the path represents amodule tree
	Path []string `json:"path"`

	// Locals are kept only transiently in-memory, because we can always
	// re-compute them.
	Locals map[string]interface{} `json:"-"`

	// Outputs declared by the module and maintained for each module
	// even though only the root module technically needs to be kept.
	// This allows operators to inspect values at the boundaries.
	Outputs map[string]*OutputState `json:"outputs"`

	// Resources is a mapping of the logically named resource to
	// the state of the resource. Each resource may actually have
	// N instances underneath, although a user only needs to think
	// about the 1:1 case.
	Resources map[string]*ResourceState `json:"resources"`

	// Dependencies are a list of things that this module relies on
	// existing to remain intact. For example: an module may depend
	// on a VPC ID given by an aws_vpc resource.
	//
	// Terraform uses this information to build valid destruction
	// orders and to warn the user if they're destroying a module that
	// another resource depends on.
	//
	// Things can be put into this list that may not be managed by
	// Terraform. If Terraform doesn't find a matching ID in the
	// overall state, then it assumes it isn't managed and doesn't
	// worry about it.
	Dependencies []string `json:"depends_on"`

	mu sync.Mutex
}

// ResourceStateKey is a structured representation of the key used for the
// ModuleState.Resources mapping
type ResourceStateKey struct {
	Name  string
	Type  string
	Mode  ResourceMode
	Index int
}

// Equal determines whether two ResourceStateKeys are the same
func (rsk *ResourceStateKey) Equal(other *ResourceStateKey) bool {
	if rsk == nil || other == nil {
		return false
	}
	if rsk.Mode != other.Mode {
		return false
	}
	if rsk.Type != other.Type {
		return false
	}
	if rsk.Name != other.Name {
		return false
	}
	if rsk.Index != other.Index {
		return false
	}
	return true
}

func (rsk *ResourceStateKey) String() string {
	if rsk == nil {
		return ""
	}
	var prefix string
	switch rsk.Mode {
	case ManagedResourceMode:
		prefix = ""
	case DataResourceMode:
		prefix = "data."
	default:
		panic(fmt.Errorf("unknown resource mode %s", rsk.Mode))
	}
	if rsk.Index == -1 {
		return fmt.Sprintf("%s%s.%s", prefix, rsk.Type, rsk.Name)
	}
	return fmt.Sprintf("%s%s.%s.%d", prefix, rsk.Type, rsk.Name, rsk.Index)
}

// ParseResourceStateKey accepts a key in the format used by
// ModuleState.Resources and returns a resource name and resource index. In the
// state, a resource has the format "type.name.index" or "type.name". In the
// latter case, the index is returned as -1.
func ParseResourceStateKey(k string) (*ResourceStateKey, error) {
	parts := strings.Split(k, ".")
	mode := ManagedResourceMode
	if len(parts) > 0 && parts[0] == "data" {
		mode = DataResourceMode
		// Don't need the constant "data" prefix for parsing
		// now that we've figured out the mode.
		parts = parts[1:]
	}
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("Malformed resource state key: %s", k)
	}
	rsk := &ResourceStateKey{
		Mode:  mode,
		Type:  parts[0],
		Name:  parts[1],
		Index: -1,
	}
	if len(parts) == 3 {
		index, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("Malformed resource state key index: %s", k)
		}
		rsk.Index = index
	}
	return rsk, nil
}

// ResourceState is a legacy representation of the state of a particular
// resource instance. It should actually be called InstanceState, but the
// bad legacy name is preserved for compatibility with existing callers.
type ResourceState struct {
	// This is filled in and managed by Terraform, and is the resource
	// type itself such as "mycloud_instance". If a resource provider sets
	// this value, it won't be persisted.
	Type string `json:"type"`

	// Dependencies are a list of things that this resource relies on
	// existing to remain intact. For example: an AWS instance might
	// depend on a subnet (which itself might depend on a VPC, and so
	// on).
	//
	// Terraform uses this information to build valid destruction
	// orders and to warn the user if they're destroying a resource that
	// another resource depends on.
	//
	// Things can be put into this list that may not be managed by
	// Terraform. If Terraform doesn't find a matching ID in the
	// overall state, then it assumes it isn't managed and doesn't
	// worry about it.
	Dependencies []string `json:"depends_on"`

	// Primary is the current active instance for this resource.
	// It can be replaced but only after a successful creation.
	// This is the instances on which providers will act.
	Primary *InstanceState `json:"primary"`

	// Deposed is used in the mechanics of CreateBeforeDestroy: the existing
	// Primary is Deposed to get it out of the way for the replacement Primary to
	// be created by Apply. If the replacement Primary creates successfully, the
	// Deposed instance is cleaned up.
	//
	// If there were problems creating the replacement Primary, the Deposed
	// instance and the (now tainted) replacement Primary will be swapped so the
	// tainted replacement will be cleaned up instead.
	//
	// An instance will remain in the Deposed list until it is successfully
	// destroyed and purged.
	Deposed []*InstanceState `json:"deposed"`

	// Provider is used when a resource is connected to a provider with an alias.
	// If this string is empty, the resource is connected to the default provider,
	// e.g. "aws_instance" goes with the "aws" provider.
	// If the resource block contained a "provider" key, that value will be set here.
	Provider string `json:"provider"`

	mu sync.Mutex
}

// InstanceState is a legacy representation of the state of a single resource
// instance object. It should really be called InstanceObjectState, but the
// bad legacy name is preserved for compatibility with existing callers.
type InstanceState struct {
	// A unique ID for this resource. This is opaque to Terraform
	// and is only meant as a lookup mechanism for the providers.
	ID string `json:"id"`

	// Attributes are basic information about the resource. Any keys here
	// are accessible in variable format within Terraform configurations:
	// ${resourcetype.name.attribute}.
	Attributes map[string]string `json:"attributes"`

	// Ephemeral is used to store any state associated with this instance
	// that is necessary for the Terraform run to complete, but is not
	// persisted to a state file.
	Ephemeral EphemeralState `json:"-"`

	// Meta is a simple K/V map that is persisted to the State but otherwise
	// ignored by Terraform core. It's meant to be used for accounting by
	// external client code. The value here must only contain Go primitives
	// and collections.
	Meta map[string]interface{} `json:"meta"`

	// Tainted is used to mark a resource for recreation.
	Tainted bool `json:"tainted"`

	mu sync.Mutex
}

// EphemeralState is a legacy type now used only to represent the resource type
// of a new resource instance object during a "terraform import" operation.
type EphemeralState struct {
	// ConnInfo is no longer used and is now ignored.
	ConnInfo map[string]string `json:"-"`

	// Type is used to specify the resource type for this instance. This is only
	// required for import operations (as documented). If the documentation
	// doesn't state that you need to set this, then don't worry about
	// setting it.
	Type string `json:"-"`
}
