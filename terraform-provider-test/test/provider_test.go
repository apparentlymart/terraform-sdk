package test

import (
	"os"
	"testing"

	tftest "github.com/apparentlymart/terraform-plugin-test"
	tfsdk "github.com/apparentlymart/terraform-sdk"
)

var testHelper *tftest.Helper

func TestMain(m *testing.M) {
	testHelper = tfsdk.InitProviderTesting("test", Provider())
	status := m.Run()
	testHelper.Close()
	os.Exit(status)
}
