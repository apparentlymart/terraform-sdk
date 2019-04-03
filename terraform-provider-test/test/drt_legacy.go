package test

import (
	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tflegacy"
)

func legacyDataResourceType() tfsdk.DataResourceType {
	return tfsdk.LegacyDataResourceType(&tflegacy.Resource{})
}
