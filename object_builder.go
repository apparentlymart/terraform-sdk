package tfsdk

import (
	"github.com/zclconf/go-cty/cty"
)

// An ObjectBuilder is a helper for gradually constructing a new value that
// conforms to a particular schema through mutation.
//
// Terraform type system values are normally immutable, but ObjectBuilder
// provides a mutable representation of an object value that can, once ready,
// be frozen into an immutable object value.
type ObjectBuilder interface {
	// ObjectBuilder extends ObjectReader, providing access to the current
	// state of the object under construction.
	//
	// Call ObjectVal for a cty.Value representation of the whole object, once
	// all mutations are complete.
	ObjectReader

	// SetAttr replaces the value of the specified attribute with the given
	// value. It will panic if the given name is not defined as an attribute
	// for this object or if the given value is not compatible with the
	// type constraint given for the attribute in the schema.
	SetAttr(name string, val cty.Value)

	// The Block... family of methods echoes the methods with similar names on
	// ObjectReader but each returns an ObjectBuilder that can be used to
	// mutate the content of the requested block.
	//
	// ObjectBuilder does not permit modifying the collection of nested blocks
	// itself, because most Terraform operations require the result to contain
	// exactly the same blocks as given in configuration.
	BlockBuilderSingle(blockType string) ObjectBuilder
	BlockBuilderList(blockType string) []ObjectBuilder
	BlockBuilderMap(blockType string) map[string]ObjectBuilder
	BlockBuilderFromList(blockType string, idx int) ObjectBuilder
	BlockBuilderFromMap(blockType string, key string) ObjectBuilder
}

// ObjectBuilderFull is an extension of ObjectBuilder that additionally allows
// totally replacing the collection of nested blocks of a given type.
//
// This interface is separate because most Terraform operations do not permit
// this change. For resource types, it is allowed only for the ReadFn
// implementation in order to synchronize the collection of nested blocks with
// the collection of corresponding objects in the remote system.
type ObjectBuilderFull interface {
	ObjectBuilder

	// NewBlockBuilder returns an ObjectBuilderFull that can construct an object
	// of a type suitable to build a new nested block of the given type. It will
	// panic if no nested block type of the given name is defined.
	//
	// The returned builder is disconnected from the object that creates it
	// in the sense that modifications won't be reflected anywhere in the
	// creator. To make use of the result, call ObjectVal to obtain an
	// object value and pass it to one of the "ReplaceBlock..." methods.
	NewBlockBuilder(blockType string) ObjectBuilderFull

	// The ReplaceBlock... family of methods remove all blocks of the given
	// type and then construct new blocks from the given object(s) in their
	// place. The given object values must conform to the nested block type
	// schema or these methods will panic. These will panic also if the
	// method used doesn't correspond with the nesting mode of the given
	// nested block type.
	ReplaceBlockSingle(blockType string, obj cty.Value)
	ReplaceBlocksList(blockType string, objs []cty.Value)
	ReplaceBlocksMap(blockType string, objs map[string]cty.Value)
}
