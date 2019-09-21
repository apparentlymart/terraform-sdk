package tftest

import (
	"os"

	"github.com/apparentlymart/terraform-sdk/tflegacy"
)

// InitProviderLegacy is similar to InitProvider but it accepts a provider
// defined using the legacy Terraform SDK.
func InitProviderLegacy(name string, providerFunc tflegacy.ProviderFunc) *Helper {
	if runningAsPlugin() {
		// The test program is being re-launched as a provider plugin via our
		// stub program.
		tflegacy.Serve(&tflegacy.ServeOpts{
			ProviderFunc: providerFunc,
		})
		os.Exit(0)
	}

	return initProviderHelper(name)
}
