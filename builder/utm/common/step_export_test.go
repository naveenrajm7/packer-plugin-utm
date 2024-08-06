//go:build exclude
// +build exclude

// we disable this test as it is not possible to test
// the export step without actually exporting the VM

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepExport_impl(t *testing.T) {
	var _ multistep.Step = new(StepExport)
}

func TestStepExport(t *testing.T) {
	state := testState(t)
	step := new(StepExport)

	state.Put("vmName", "foo")
	// We use the commHostPort to clear the forwarded ports
	state.Put("commHostPort", 1234)
	// driver := state.Get("driver").(*DriverMock)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test output state
	if _, ok := state.GetOk("exportPath"); !ok {
		t.Fatal("should set exportPath")
	}

	// Test driver
	// TODO: Test calls to driver
	// Currently our export step doesn't call the driver
	// It asks for manual intervention to export the VM
	// and just sets the exportPath in the state bag
}

func TestStepExport_OutputPath(t *testing.T) {
	type testCase struct {
		Step     *StepExport
		Expected string
		Reason   string
	}
	tcs := []testCase{
		{
			Step: &StepExport{
				Format:         "ova",
				OutputDir:      "output-dir",
				OutputFilename: "output-filename",
			},
			Expected: filepath.Join("output-dir", "output-filename.ova"),
			Reason:   "output_filename should not be vmName if set.",
		},
		{
			Step: &StepExport{
				Format:         "ovf",
				OutputDir:      "output-dir",
				OutputFilename: "",
			},
			Expected: filepath.Join("output-dir", "foo.ovf"),
			Reason:   "output_filename should default to vmName.",
		},
	}
	for _, tc := range tcs {
		state := testState(t)
		state.Put("vmName", "foo")
		// We use the commHostPort to clear the forwarded ports
		state.Put("commHostPort", 1234)
		// Test the run
		if action := tc.Step.Run(context.Background(), state); action != multistep.ActionContinue {
			t.Fatalf("bad action: %#v", action)
		}

		// Test output state
		path, ok := state.GetOk("exportPath")
		if !ok {
			t.Fatal("should set exportPath")
		}
		if path != tc.Expected {
			t.Fatalf("Expected %s didn't match received %s: %s", tc.Expected, path, tc.Reason)
		}
	}
}

func TestStepExport_SkipExport(t *testing.T) {
	state := testState(t)
	step := StepExport{SkipExport: true}

	state.Put("vmName", "foo")
	// We use the commHostPort to clear the forwarded ports
	state.Put("commHostPort", 1234)
	// driver := state.Get("driver").(*DriverMock)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}
	// Test driver
	// TODO: Test calls to driver
	// Currently our export step doesn't call the driver
	// It asks for manual intervention to export the VM
	// and just sets the exportPath in the state bag

}
