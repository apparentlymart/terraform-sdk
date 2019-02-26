package test

import (
	"fmt"
	"net/url"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/zclconf/go-cty/cty"
)

func Provider() *tfsdk.Provider {
	return &tfsdk.Provider{
		ConfigSchema: &tfsdk.SchemaBlockType{
			Attributes: map[string]*tfsdk.SchemaAttribute{
				"optional_string": {
					Type:     cty.String,
					Optional: true,
				},
				"optional_url": {
					Type:     cty.String,
					Optional: true,
					ValidateFn: func(v string) tfsdk.Diagnostics {
						var diags tfsdk.Diagnostics
						_, err := url.Parse(v)
						if err != nil {
							diags = diags.Append(tfsdk.Diagnostic{
								Severity: tfsdk.Error,
								Summary:  "Invalid URL",
								Detail:   fmt.Sprintf("A URL is required: %s.", err.Error()),
							})
						}
						return diags
					},
				},
			},
		},
	}
}
