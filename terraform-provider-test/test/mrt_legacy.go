package test

import (
	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tflegacy"
)

func legacyManagedResourceType() tfsdk.ManagedResourceType {
	return tfsdk.LegacyManagedResourceType(&tflegacy.Resource{})
}
