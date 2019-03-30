package test

import (
	"context"
	"log"
	"net/url"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"
)

func Provider() *tfsdk.Provider {
	return &tfsdk.Provider{
		ConfigSchema: &tfschema.BlockType{
			Attributes: map[string]*tfschema.Attribute{
				"optional_string": {
					Type:     cty.String,
					Optional: true,
				},
				"optional_url": {
					Type:     cty.String,
					Optional: true,
					ValidateFn: func(v string) tfsdk.Diagnostics {
						var diags tfsdk.Diagnostics
						u, err := url.Parse(v)
						if err != nil || u.Scheme != "https" {
							diags = diags.Append(tfsdk.Diagnostic{
								Severity: tfsdk.Error,
								Summary:  "Invalid URL",
								Detail:   "Must be a valid absolute HTTPS URL.",
							})
						}
						return diags
					},
				},
			},
		},
		ConfigureFn: func(ctx context.Context, config *Config) (*Client, tfsdk.Diagnostics) {
			var diags tfsdk.Diagnostics
			log.Printf("test provider configured with %s", spew.Sdump(config))
			return &Client{}, diags
		},

		ManagedResourceTypes: map[string]tfsdk.ManagedResourceType{
			"test_instance": instanceManagedResourceType(),
		},
		DataResourceTypes: map[string]tfsdk.DataResourceType{
			"test_echo": echoDataResourceType(),
		},
	}
}

type Config struct {
	OptionalString *string `cty:"optional_string"`
	OptionalURL    *string `cty:"optional_url"`
}

type Client struct {
}
