package test

import (
	"testing"
)

func TestMRTInstance(t *testing.T) {
	wd := testHelper.RequireNewWorkingDir(t)
	defer wd.Close()

	wd.RequireSetConfig(t, `
resource "test_instance" "test" {
  type  = "z2.wheezy"
  image = "img-abc123"

  access {
    policy = {}
  }

  network_interface "main" {
    create_public_addrs = false
  }
  network_interface "public" {
    create_public_addrs = true
  }
}

output "id" {
  value = test_instance.test.id
}
`)

	wd.RequireInit(t)
	wd.RequireApply(t)
}
