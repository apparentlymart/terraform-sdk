package test

import (
	"os"
	"testing"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tftest"
)

var testHelper *tftest.Helper

func TestMain(m *testing.M) {
	testHelper = tfsdk.InitProviderTesting("test", Provider())
	status := m.Run()
	testHelper.Close()
	os.Exit(status)
}
