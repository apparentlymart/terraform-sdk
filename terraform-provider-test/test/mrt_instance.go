package test

import (
	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/zclconf/go-cty/cty"
)

func instanceManagedResourceType() tfsdk.ManagedResourceType {
	return tfsdk.NewManagedResourceType(&tfsdk.ResourceType{
		ConfigSchema: &tfsdk.SchemaBlockType{
			Attributes: map[string]*tfsdk.SchemaAttribute{
				"type":  {Type: cty.String, Required: true},
				"image": {Type: cty.String, Required: true},
			},
			NestedBlockTypes: map[string]*tfsdk.SchemaNestedBlockType{
				"network_interface": {
					Nesting: tfsdk.SchemaNestingMap,
					Content: tfsdk.SchemaBlockType{
						Attributes: map[string]*tfsdk.SchemaAttribute{
							"create_public_addrs": {Type: cty.Bool, Optional: true},
						},
					},
				},
			},
		},
	})
}
