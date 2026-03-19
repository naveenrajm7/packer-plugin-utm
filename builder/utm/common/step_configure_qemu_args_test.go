// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepConfigureQemuArgs_impl(t *testing.T) {
	var _ multistep.Step = new(StepConfigureQemuArgs)
}

func TestStepConfigureQemuArgs_noArgs(t *testing.T) {
	state := testState(t)
	state.Put("vmId", "test-vm-id")

	step := &StepConfigureQemuArgs{
		QemuArgs: nil,
	}

	action := step.Run(context.Background(), state)
	if action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Verify no driver calls were made
	driver := state.Get("driver").(*DriverMock)
	if len(driver.ExecuteOsaCalls) > 0 {
		t.Fatalf("should not have called ExecuteOsaScript, got: %#v", driver.ExecuteOsaCalls)
	}
}

func TestStepConfigureQemuArgs_withArgs(t *testing.T) {
	state := testState(t)
	state.Put("vmId", "test-vm-id")

	step := &StepConfigureQemuArgs{
		QemuArgs: [][]string{
			{"-accel", "hvf"},
			{"-cpu", "host"},
		},
	}

	action := step.Run(context.Background(), state)
	if action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Verify the driver was called with correct args
	driver := state.Get("driver").(*DriverMock)
	if len(driver.ExecuteOsaCalls) != 1 {
		t.Fatalf("expected 1 ExecuteOsaScript call, got %d", len(driver.ExecuteOsaCalls))
	}

	call := driver.ExecuteOsaCalls[0]
	expected := []string{
		"add_qemu_additional_args.applescript", "test-vm-id",
		"--args", "-accel hvf", "-cpu host",
	}
	if len(call) != len(expected) {
		t.Fatalf("expected %d args, got %d: %#v", len(expected), len(call), call)
	}
	for i, arg := range expected {
		if call[i] != arg {
			t.Fatalf("arg[%d]: expected %q, got %q", i, arg, call[i])
		}
	}
}

func TestStepConfigureQemuArgs_driverError(t *testing.T) {
	state := testState(t)
	state.Put("vmId", "test-vm-id")

	driver := state.Get("driver").(*DriverMock)
	driver.ExecuteOsaErrs = []error{fmt.Errorf("applescript failed")}

	step := &StepConfigureQemuArgs{
		QemuArgs: [][]string{
			{"-cpu", "host"},
		},
	}

	action := step.Run(context.Background(), state)
	if action != multistep.ActionHalt {
		t.Fatalf("expected ActionHalt, got: %#v", action)
	}
	if _, ok := state.GetOk("error"); !ok {
		t.Fatal("should have error")
	}
}
