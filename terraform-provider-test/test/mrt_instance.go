package test

import (
	tfsdk "github.com/apparentlymart/terraform-sdk"
)

func instanceManagedResourceType() tfsdk.ManagedResourceType {
	return tfsdk.NewManagedResourceType(&tfsdk.ResourceType{})
}
