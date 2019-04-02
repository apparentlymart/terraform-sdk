package tflegacy

type ResourceMode int

//go:generate stringer -type=ResourceMode -output=resource_mode_string.go config.go

const (
	ManagedResourceMode ResourceMode = iota
	DataResourceMode
)

// ResourceConfig is a legacy type that was formerly used to represent
// interpolatable configuration blocks. It is now only used to shim to old
// APIs that still use this type, via NewResourceConfigShimmed.
type ResourceConfig struct {
	ComputedKeys []string
	Raw          map[string]interface{}
	Config       map[string]interface{}
}
