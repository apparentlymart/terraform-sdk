package test

import (
	"testing"
)

func TestMRTLegacy(t *testing.T) {
	wd := testHelper.RequireNewWorkingDir(t)
	defer wd.Close()

	wd.RequireSetConfig(t, `
resource "test_legacy" "test" {
}
`)

	wd.RequireInit(t)
	wd.RequireApply(t)
}
