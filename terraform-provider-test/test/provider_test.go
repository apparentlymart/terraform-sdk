package test

import (
	"os"
	"testing"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	tftest "github.com/apparentlymart/terraform-plugin-test"
)

var testHelper *tftest.Helper

func TestMain(m *testing.M) {
	testHelper = tfsdk.InitProviderTesting("test", Provider())
	status := m.Run()
	testHelper.Close()
	os.Exit(status)
}
