// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

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
	// TODO: If ISO export is configured, ensure this option is propagated to UTM.
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
	ui.Say("Preparing to export machine...")

	// Clear out the Packer-created forwarding rule
	commPort := state.Get("commHostPort")
	if !s.SkipNatMapping && commPort != 0 {
		ui.Message(fmt.Sprintf(
			"Deleting forwarded port mapping for the communicator (SSH, WinRM, etc) (host port %d)", commPort))
		// Assert that commPort is of type int
		commPortInt, ok := commPort.(int)
		if !ok {
			err := fmt.Errorf("commPort is not of type int")
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		command := []string{
			"clear_port_forwards.applescript", vmName,
			"--index", "1", strconv.Itoa(commPortInt),
		}
		if _, err := driver.ExecuteOsaScript(command...); err != nil {
			err := fmt.Errorf("error deleting port forwarding rule: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Export the VM to an UTM file
	outputPath := filepath.Join(s.OutputDir, s.OutputFilename+"."+s.Format)
	ui.Say("Exporting virtual machine...")

	// TODO: Actually export the VM when UTM API supports it
	// Till then ask the user to manually export the VM
	// using Share action in UTM VM in output Path
	ui.Say("UTM API does not support exporting VMs yet.")
	ui.Say("Please manually export the VM using 'Share...' action in UTM VM menu.")
	ui.Say(fmt.Sprintf("Please make sure the VM is exported to the path %s ", outputPath))
	ui.Say("The exported UTM file in the output directory will be passed as build Artifact.")
	// ask user to input the path of the exported file
	confirmOption, err := ui.Ask(
		fmt.Sprintf("Confirm you have exported the VM to path [%s] [Y/n]:", outputPath))

	if err != nil {
		err := fmt.Errorf("error during export step: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if confirmOption == "Y" || confirmOption == "y" {
		// Proceed with the next steps
		ui.Say("Proceeding assuming the export is done...")
		// We set export path as the output directory with UTM file.
		// So it can be used as an artifact in the next steps.
		state.Put("exportPath", outputPath)

		return multistep.ActionContinue
	} else {
		ui.Say("Export halted by user.")
		return multistep.ActionHalt
	}

}

func (s *StepExport) Cleanup(state multistep.StateBag) {}
