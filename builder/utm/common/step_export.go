// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step cleans up forwarded ports and (TODO) exports the VM to an UTM file.
//
// Uses:
//
// Produces:
//
//	exportPath string - The path to the resulting export.
type StepExport struct {
	Format         string
	OutputDir      string
	OutputFilename string
	ExportOpts     []string
	Bundling       UtmBundleConfig
	SkipNatMapping bool
	SkipExport     bool
}

func (s *StepExport) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	// If ISO export is configured, ensure this option is propagated to UTM.
	for _, option := range s.ExportOpts {
		if option == "--iso" || option == "-I" {
			s.ExportOpts = append(s.ExportOpts, "--iso")
			break
		}
	}

	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmName := state.Get("vmName").(string)
	if s.OutputFilename == "" {
		s.OutputFilename = vmName
	}

	// Skip export if requested
	if s.SkipExport {
		ui.Say("Skipping export of virtual machine...")
		return multistep.ActionContinue
	}
	// TODO: Actually export the VM
	ui.Say("Preparing to export machine...")

	// Clear out the Packer-created forwarding rule
	commPort := state.Get("commHostPort")
	if !s.SkipNatMapping && commPort != 0 {
		ui.Message(fmt.Sprintf(
			"Deleting forwarded port mapping for the communicator (SSH, WinRM, etc) (host port %d)", commPort))
		command := []string{
			"clear_port_forwards.applescript", vmName,
			"--index", "1", commPort.(string),
		}
		if _, err := driver.Utmctl(command...); err != nil {
			err := fmt.Errorf("error deleting port forwarding rule: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Export the VM to an UTM file
	outputPath := filepath.Join(s.OutputDir, s.OutputFilename+"."+s.Format)
	ui.Say("Exporting virtual machine...")

	state.Put("exportPath", outputPath)

	return multistep.ActionContinue
}

func (s *StepExport) Cleanup(state multistep.StateBag) {}
