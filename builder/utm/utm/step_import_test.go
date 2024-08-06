// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utm

import (
	"context"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

func TestStepImport_impl(t *testing.T) {
	var _ multistep.Step = new(StepImport)
}

func TestStepImport(t *testing.T) {
	state := testState(t)
	state.Put("vm_path", "foo")

	step := new(StepImport)
	step.Name = "bar"

	driver := state.Get("driver").(*utmcommon.DriverMock)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test driver
	if !driver.ImportCalled {
		t.Fatal("import should be called")
	}
	if driver.ImportName != step.Name {
		t.Fatalf("bad: %#v", driver.ImportName)
	}

	// Test output state
	if name, ok := state.GetOk("vmName"); !ok {
		t.Fatal("vmName should be set")
	} else if name != "bar" {
		t.Fatalf("bad: %#v", name)
	}
}

func TestStepImport_Cleanup(t *testing.T) {
	state := testState(t)
	state.Put("vm_path", "foo")

	step := new(StepImport)
	step.vmName = "bar"

	driver := state.Get("driver").(*utmcommon.DriverMock)

	step.KeepRegistered = true
	step.Cleanup(state)
	if driver.DeleteCalled {
		t.Fatal("delete should not be called")
	}

	state.Put(multistep.StateHalted, true)
	step.Cleanup(state)
	if !driver.DeleteCalled {
		t.Fatal("delete should be called")
	}
	if driver.DeleteName != "bar" {
		t.Fatalf("bad: %#v", driver.DeleteName)
	}
}
