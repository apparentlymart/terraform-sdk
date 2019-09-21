package tflegacy

import (
	oldsdk "github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// Schema represents the description of a single attribute or block type in
// the legacy SDK.
type Schema = oldsdk.Schema

// ValueType represents a value type in the legacy SDK.
type ValueType = oldsdk.ValueType

const (
	TypeBool   = oldsdk.TypeBool
	TypeInt    = oldsdk.TypeInt
	TypeFloat  = oldsdk.TypeFloat
	TypeString = oldsdk.TypeString
	TypeList   = oldsdk.TypeList
	TypeSet    = oldsdk.TypeSet
	TypeMap    = oldsdk.TypeMap
)

type SchemaConfigMode = oldsdk.SchemaConfigMode

const (
	SchemaConfigModeAttr  = oldsdk.SchemaConfigModeAttr
	SchemaConfigModeBlock = oldsdk.SchemaConfigModeBlock
)
