package tflegacy

type ResourceMode int

//go:generate stringer -type=ResourceMode -output=resource_mode_string.go config.go

const (
	ManagedResourceMode ResourceMode = iota
	DataResourceMode
)
