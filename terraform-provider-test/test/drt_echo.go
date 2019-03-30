package test

import (
	"context"
	"log"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/zclconf/go-cty/cty"
)

type echoDRT struct {
	Given  string  `cty:"given"`
	Result *string `cty:"result"`

	Dynamic cty.Value `cty:"dynamic"`
}

func echoDataResourceType() tfsdk.DataResourceType {
	return tfsdk.NewDataResourceType(&tfsdk.ResourceType{
		ConfigSchema: &tfschema.BlockType{
			Attributes: map[string]*tfschema.Attribute{
				"given":  {Type: cty.String, Required: true},
				"result": {Type: cty.String, Computed: true},

				"dynamic": {Type: cty.DynamicPseudoType, Optional: true},
			},
		},

		ReadFn: func(cty context.Context, client *Client, obj *echoDRT) (*echoDRT, tfsdk.Diagnostics) {
			log.Printf("reading %#v", obj)
			obj.Result = &obj.Given
			return obj, nil
		},
	})
}
