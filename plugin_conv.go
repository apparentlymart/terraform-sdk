package tfsdk

import (
	"fmt"
	"sort"

	"github.com/apparentlymart/terraform-sdk/internal/tfplugin5"
)

func convertSchemaBlockToTFPlugin5(src *SchemaBlockType) *tfplugin5.Schema_Block {
	ret := &tfplugin5.Schema_Block{}
	if src == nil {
		// Weird, but we'll allow it.
		return ret
	}

	for name, attrS := range src.Attributes {
		tyJSON, err := attrS.Type.MarshalJSON()
		if err != nil {
			// Should never happen, since types should always be valid
			panic(fmt.Sprintf("failed to serialize %#v as JSON: %s", attrS.Type, err))
		}
		ret.Attributes = append(ret.Attributes, &tfplugin5.Schema_Attribute{
			Name:        name,
			Type:        tyJSON,
			Description: attrS.Description,
			Required:    attrS.Required,
			Optional:    attrS.Optional,
			Computed:    attrS.Computed,
			Sensitive:   attrS.Sensitive,
		})
	}

	for name, blockS := range src.NestedBlockTypes {
		nested := convertSchemaBlockToTFPlugin5(&blockS.Content)
		var nesting tfplugin5.Schema_NestedBlock_NestingMode
		switch blockS.Nesting {
		case SchemaNestingSingle:
			nesting = tfplugin5.Schema_NestedBlock_SINGLE
		case SchemaNestingList:
			nesting = tfplugin5.Schema_NestedBlock_LIST
		case SchemaNestingMap:
			nesting = tfplugin5.Schema_NestedBlock_MAP
		case SchemaNestingSet:
			nesting = tfplugin5.Schema_NestedBlock_SET
		default:
			// Should never happen because the above is exhaustive.
			panic(fmt.Sprintf("unsupported block nesting mode %#v", blockS.Nesting))
		}
		ret.BlockTypes = append(ret.BlockTypes, &tfplugin5.Schema_NestedBlock{
			TypeName: name,
			Nesting:  nesting,
			Block:    nested,
			MaxItems: int64(blockS.MaxItems),
			MinItems: int64(blockS.MinItems),
		})
	}

	sort.Slice(ret.Attributes, func(i, j int) bool {
		return ret.Attributes[i].Name < ret.Attributes[j].Name
	})

	return ret
}
