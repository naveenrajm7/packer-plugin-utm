// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step TODO:(cleans up forwarded ports) and exports the VM to an UTM.
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
	// If ISO export is configured, ensure this option is propagated to VBoxManage.
	for _, option := range s.ExportOpts {
		if option == "--iso" || option == "-I" {
			s.ExportOpts = append(s.ExportOpts, "--iso")
			break
		}
	}

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

	// Export the VM to an OVF
	outputPath := filepath.Join(s.OutputDir, s.OutputFilename+"."+s.Format)
	ui.Say("Exporting virtual machine...")

	state.Put("exportPath", outputPath)

	return multistep.ActionContinue
}

func (s *StepExport) Cleanup(state multistep.StateBag) {}
