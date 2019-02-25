package main

import (
	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/terraform-provider-test/test"
)

func main() {
	tfsdk.ServeProviderPlugin(test.Provider())
}
