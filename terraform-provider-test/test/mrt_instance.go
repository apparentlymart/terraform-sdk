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

	Access            *instanceMRTAccess                      `cty:"access"`
	NetworkInterfaces map[string]*instanceMRTNetworkInterface `cty:"network_interface"`
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

		ReadFn: func(cty context.Context, client *Client, current *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("reading %#v", current)
			return current, nil // No changes
		},
		CreateFn: func(ctx context.Context, client *Client, new *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("creating %#v", new)
			id := "placeholder"
			new.ID = &id
			return new, nil
		},
		UpdateFn: func(ctx context.Context, client *Client, prior, new *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("updating %#v", new)
			return new, nil
		},
		DeleteFn: func(ctx context.Context, client *Client, prior *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("deleting %#v", prior)
			return nil, nil
		},
	})
}
