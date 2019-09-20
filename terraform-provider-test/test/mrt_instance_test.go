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

	t.Log("init")
	wd.RequireInit(t)

	t.Log("create initial plan")
	wd.RequireCreatePlan(t)
	t.Log("read initial plan")
	plan := wd.RequireSavedPlan(t)
	if got, want := len(plan.PlannedValues.RootModule.Resources), 1; got != want {
		t.Fatalf("wrong number of planned resource changes %d; want %d", got, want)
	}
	plannedChange := plan.PlannedValues.RootModule.Resources[0]
	if got, want := plannedChange.Type, "test_instance"; got != want {
		t.Errorf("wrong resource type in plan\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := plannedChange.AttributeValues["version"], 1.0; got != want {
		// All numbers are decoded as float64 by RequireSavedPlan, because
		// Terraform itself does not distinguish number types.
		t.Errorf("wrong 'version' value in create plan\ngot:  %#v (%T)\nwant: %#v (%T)", got, got, want, want)
	}

	t.Log("apply initial plan")
	wd.RequireApply(t)

	t.Log("read state after initial apply")
	state := wd.RequireState(t)
	outputs := state.Values.Outputs
	idOutput := outputs["id"]
	if idOutput == nil {
		t.Fatal("missing 'id' output")
	}
	if got, want := idOutput.Value, "placeholder"; got != want {
		t.Errorf("wrong value for id output\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := len(state.Values.RootModule.Resources), 1; got != want {
		t.Fatalf("wrong number of resource instance objects in state %d; want %d", got, want)
	}
	instanceState := state.Values.RootModule.Resources[0]
	if got, want := instanceState.Type, "test_instance"; got != want {
		t.Errorf("wrong resource type in state\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := instanceState.AttributeValues["version"], 1.0; got != want {
		// All numbers are decoded as float64 by RequireSavedPlan, because
		// Terraform itself does not distinguish number types.
		t.Errorf("wrong 'version' value\ngot:  %#v (%T)\nwant: %#v (%T)", got, got, want, want)
	}

	initialID := idOutput.Value

	// Update existing object
	wd.RequireSetConfig(t, `
resource "test_instance" "test" {
  type  = "z2.wheezy"
  image = "img-abc456" # image has changed

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

	t.Log("create followup plan")
	wd.RequireCreatePlan(t)
	t.Log("read followup plan")
	plan = wd.RequireSavedPlan(t)
	if got, want := len(plan.PlannedValues.RootModule.Resources), 1; got != want {
		t.Fatalf("wrong number of planned resource changes %d; want %d", got, want)
	}
	plannedChange = plan.PlannedValues.RootModule.Resources[0]
	if got, want := plannedChange.Type, "test_instance"; got != want {
		t.Errorf("wrong resource type in update plan\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := plannedChange.AttributeValues["version"], (interface{})(nil); got != want {
		// Version should not be present because we changed "image" and so now
		// it is unknown until after apply.
		t.Errorf("wrong 'version' value in update plan\ngot:  %#v (%T)\nwant: %#v (%T)", got, got, want, want)
	}

	t.Log("apply followup plan")
	wd.RequireApply(t)

	state = wd.RequireState(t)
	t.Log("read state after followup apply")
	outputs = state.Values.Outputs
	idOutput = outputs["id"]
	if idOutput == nil {
		t.Fatal("missing 'id' output")
	}
	if got, want := idOutput.Value, initialID; got != want {
		t.Errorf("wrong value for id output after update\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := len(state.Values.RootModule.Resources), 1; got != want {
		t.Fatalf("wrong number of resource instance objects in state %d; want %d", got, want)
	}
	instanceState = state.Values.RootModule.Resources[0]
	if got, want := instanceState.Type, "test_instance"; got != want {
		t.Errorf("wrong resource type in state after update\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := instanceState.AttributeValues["version"], 2.0; got != want {
		// Version should've been incremented because the 'image' changed
		t.Errorf("wrong 'version' value after update\ngot:  %#v (%T)\nwant: %#v (%T)", got, got, want, want)
	}

}
