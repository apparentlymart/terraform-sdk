package test

import (
	"context"
	"fmt"
	"log"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tfobj"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/zclconf/go-cty/cty"
)

type instanceMRT struct {
	ID      *string `cty:"id"`
	Version *int    `cty:"version"`
	Type    string  `cty:"type"`
	Image   string  `cty:"image"`

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
	return tfsdk.NewManagedResourceType(&tfsdk.ResourceTypeDef{
		ConfigSchema: &tfschema.BlockType{
			Attributes: map[string]*tfschema.Attribute{
				"id":      {Type: cty.String, Computed: true},
				"version": {Type: cty.Number, Computed: true},

				"type":  {Type: cty.String, Required: true},
				"image": {Type: cty.String, Required: true},
			},
			NestedBlockTypes: map[string]*tfschema.NestedBlockType{
				"network_interface": {
					Nesting: tfschema.NestingMap,
					Content: tfschema.BlockType{
						Attributes: map[string]*tfschema.Attribute{
							"create_public_addrs": {
								Type:     cty.Bool,
								Optional: true,
								Default:  true,
							},
						},
					},
				},
				"access": {
					Nesting: tfschema.NestingSingle,
					Content: tfschema.BlockType{
						Attributes: map[string]*tfschema.Attribute{
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

		PlanFn: func(ctx context.Context, client *Client, plan tfobj.PlanBuilder) (cty.Value, tfsdk.Diagnostics) {
			prior, planned := plan.AttrChange("type")
			log.Printf("'type' value was %#v and is now %#v", prior, planned)
			switch plan.Action() {
			case tfobj.Create:
				plan.SetAttr("version", cty.NumberIntVal(1))
			default:
				if plan.AttrHasChange("type") || plan.AttrHasChange("image") {
					plan.SetAttrUnknown("version") // we'll allocate a new version at apply time
				}
			}
			return plan.ObjectVal(), nil
		},
		ReadFn: func(ctx context.Context, client *Client, current *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("reading %#v", current)
			return current, nil // No changes
		},
		CreateFn: func(ctx context.Context, client *Client, new *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("creating %#v", new)
			id := "placeholder"
			version := 1
			new.ID = &id
			new.Version = &version
			return new, nil
		},
		UpdateFn: func(ctx context.Context, client *Client, prior, new *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("updating %#v", new)
			if new.Version == nil {
				newVersion := 1
				if prior.Version != nil {
					newVersion = *prior.Version + 1
				}
				new.Version = &newVersion
			}
			return new, nil
		},
		DeleteFn: func(ctx context.Context, client *Client, prior *instanceMRT) (*instanceMRT, tfsdk.Diagnostics) {
			log.Printf("deleting %#v", prior)
			return nil, nil
		},
	})
}
