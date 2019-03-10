package test

import (
	"context"
	"fmt"
	"log"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/zclconf/go-cty/cty"
)

type instanceMRT struct {
	ID    *string `cty:"id"`
	Type  string  `cty:"type"`
	Image string  `cty:"image"`

	Access            *instanceMRTAccess             `cty:"access"`
	NetworkInterfaces []*instanceMRTNetworkInterface `cty:"network_interface"`
}

type instanceMRTNetworkInterface struct {
	CreatePublicAddrs bool `cty:"create_public_addrs"`
}

type instanceMRTAccess struct {
	Policy cty.Value `cty:"policy"`
}

func instanceManagedResourceType() tfsdk.ManagedResourceType {
	return tfsdk.NewManagedResourceType(&tfsdk.ResourceType{
		ConfigSchema: &tfsdk.SchemaBlockType{
			Attributes: map[string]*tfsdk.SchemaAttribute{
				"id": {Type: cty.String, Computed: true},

				"type":  {Type: cty.String, Required: true},
				"image": {Type: cty.String, Required: true},
			},
			NestedBlockTypes: map[string]*tfsdk.SchemaNestedBlockType{
				"network_interface": {
					Nesting: tfsdk.SchemaNestingMap,
					Content: tfsdk.SchemaBlockType{
						Attributes: map[string]*tfsdk.SchemaAttribute{
							"create_public_addrs": {
								Type:     cty.Bool,
								Optional: true,
								Default:  true,
							},
						},
					},
				},
				"access": {
					Nesting: tfsdk.SchemaNestingSingle,
					Content: tfsdk.SchemaBlockType{
						Attributes: map[string]*tfsdk.SchemaAttribute{
							"policy": {
								Type:     cty.DynamicPseudoType,
								Required: true,

								ValidateFn: func(val cty.Value) tfsdk.Diagnostics {
									var diags tfsdk.Diagnostics
									if !(val.Type().IsObjectType() || val.Type().IsMapType()) {
										diags = diags.Append(
											tfsdk.ValidationError(fmt.Errorf("must be an object, using { ... } syntax")),
										)
									}
									return diags
								},
							},
						},
					},
				},
			},
		},

		CreateFn: func(ctx context.Context, new *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("creating %#v", new)
			return new, nil
		},
	})
}
