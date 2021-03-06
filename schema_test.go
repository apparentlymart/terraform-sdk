package tfsdk_test

import (
	"strings"
	"testing"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestValidateAttrValue(t *testing.T) {
	tests := map[string]struct {
		Schema    *tfschema.Attribute
		Try       cty.Value
		WantDiags []string
	}{
		"simple primitive type ok": {
			&tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			},
			cty.StringVal("ok"),
			nil,
		},
		"missing required argument": {
			&tfschema.Attribute{
				Type:     cty.String,
				Required: true,
			},
			cty.NullVal(cty.String),
			[]string{
				`[ERROR] Missing required argument: This argument is required.`,
			},
		},
		"missing optional argument": {
			&tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			},
			cty.NullVal(cty.String),
			nil,
		},
		"simple primitive type conversion ok": {
			&tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
			},
			cty.True, // can become the string "true", so okay
			nil,
		},
		"simple primitive type conversion fail": {
			&tfschema.Attribute{
				Type:     cty.Bool,
				Optional: true,
			},
			cty.StringVal("not a bool"),
			[]string{
				`[ERROR] Invalid argument value: Incorrect value type: a bool is required.`,
			},
		},
		"object type missing attribute": {
			&tfschema.Attribute{
				Type: cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}),
				Optional: true,
			},
			cty.EmptyObjectVal,
			[]string{
				`[ERROR] Invalid argument value: Incorrect value type: attribute "foo" is required.`,
			},
		},
		"custom validate function ok": {
			&tfschema.Attribute{
				Type:     cty.String,
				Required: true,
				ValidateFn: func(v string) tfsdk.Diagnostics {
					if v != "ok" {
						return tfsdk.Diagnostics{
							{
								Severity: tfsdk.Error,
								Summary:  "Not ok",
							},
						}
					}
					return nil
				},
			},
			cty.StringVal("ok"),
			nil,
		},
		"custom validate function wrong": {
			&tfschema.Attribute{
				Type:     cty.String,
				Required: true,
				ValidateFn: func(v string) tfsdk.Diagnostics {
					if v != "ok" {
						return tfsdk.Diagnostics{
							{
								Severity: tfsdk.Error,
								Summary:  "Not ok",
							},
						}
					}
					return nil
				},
			},
			cty.StringVal("not ok"),
			[]string{
				`[ERROR] Not ok`,
			},
		},
		"custom validate function type conversion error": {
			&tfschema.Attribute{
				Type:     cty.String,
				Required: true,
				// This is not something any provider should really do, but
				// we want to make sure it produces a reasonable result.
				ValidateFn: func(v bool) tfsdk.Diagnostics {
					return nil
				},
			},
			cty.StringVal("not a bool"),
			[]string{
				`[ERROR] Unsuitable argument value: This value cannot be used: bool value is required.`,
			},
		},
		"custom validate function type with incorrect return type": {
			&tfschema.Attribute{
				Type:     cty.String,
				Optional: true,
				ValidateFn: func(string) string {
					return ""
				},
			},
			cty.StringVal("ok"),
			[]string{
				"[ERROR] Invalid provider schema: Invalid ValidateFn: must return Diagnostics.\nThis is a bug in the provider that should be reported in its own issue tracker.",
			},
		},
		"custom validate function type with no return type": {
			&tfschema.Attribute{
				Type:       cty.String,
				Optional:   true,
				ValidateFn: func(string) {},
			},
			cty.StringVal("ok"),
			[]string{
				"[ERROR] Invalid provider schema: Invalid ValidateFn: must return Diagnostics.\nThis is a bug in the provider that should be reported in its own issue tracker.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotDiags := tfsdk.ValidateAttrValue(test.Schema, test.Try)

			if len(test.WantDiags) > 0 {
				gotDiagsStr := diagnosticStringsForTests(gotDiags)
				if !cmp.Equal(gotDiagsStr, test.WantDiags) {
					t.Fatalf("wrong diagnostics\n%s", cmp.Diff(test.WantDiags, gotDiagsStr))
				}
				return
			}

			for _, diagStr := range diagnosticStringsForTests(gotDiags) {
				t.Errorf("unexpected problem: %s", diagStr)
			}
		})
	}
}

// diagnosticStringForTests converts a diagnostic into a compact string that
// is easier to use for matching in test assertions.
func diagnosticStringForTests(diag tfsdk.Diagnostic) string {
	var buf strings.Builder
	switch diag.Severity {
	case tfsdk.Error:
		buf.WriteString("[ERROR] ")
	case tfsdk.Warning:
		buf.WriteString("[WARNING] ")
	default:
		buf.WriteString("[???] ")
	}
	buf.WriteString(diag.Summary)
	if diag.Detail != "" {
		buf.WriteString(": ")
		buf.WriteString(diag.Detail)
	}
	if len(diag.Path) != 0 {
		buf.WriteString(" (in ")
		buf.WriteString(tfsdk.FormatPath(diag.Path))
		buf.WriteString(")")
	}
	return buf.String()
}

func diagnosticStringsForTests(diags tfsdk.Diagnostics) []string {
	ret := make([]string, len(diags))
	for i, diag := range diags {
		ret[i] = diagnosticStringForTests(diag)
	}
	return ret
}
