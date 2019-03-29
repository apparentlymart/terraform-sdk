package tfsdk

import (
	"github.com/zclconf/go-cty/cty"
)

// An ObjectReader has methods to read data from a value that conforms to a
// particular schema, such as a resource type configuration.
type ObjectReader interface {
	// Schema returns the schema that the object conforms to. Do not modify
	// any part of the returned schema.
	Schema() *SchemaBlockType

	// ObjectVal returns the whole object that the ObjectReader is providing
	// access to. The result has a type that conforms to the reader's schema.
	ObjectVal() cty.Value

	// Attr returns the value for the attribute of the given name. It will
	// panic if the given name is not defined as an attribute for this object
	// in its schema.
	Attr(name string) cty.Value

	// The "Block..." family of methods all interact with nested blocks.
	//
	// BlockSingle, BlockList, and BlockMap allow reading all of the blocks of
	// a particular type, with each one being appropriate for a different
	// SchemaNestingMode. These methods will panic if the method called isn't
	// compatible with the nesting mode. (BlockList can be used with NestingSet).
	//
	// BlockFromList and BlockFromMap similarly allow extracting a single nested
	// block from a collection of blocks of a particular type using a suitable
	// key. BlockFromList can be used only with NestingList block types and
	// BlockFromMap only with NestingMap block types. Neither method can be
	// used with NestingSet block types because set elements do not have keys.
	// These methods will panic if used with an incompatible block type.
	BlockSingle(blockType string) ObjectReader
	BlockList(blockType string) []ObjectReader
	BlockMap(blockType string) map[string]ObjectReader
	BlockFromList(blockType string, idx int) ObjectReader
	BlockFromMap(blockType string, key string) ObjectReader
}
