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
	vmId := state.Get("vmId").(string)
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
		ui.Say(fmt.Sprintf(
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
			"clear_port_forwards.applescript", vmId,
			"--index", "1", strconv.Itoa(commPortInt),
		}
		if _, err := driver.ExecuteOsaScript(command...); err != nil {
			err := fmt.Errorf("error deleting port forwarding rule: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Clear out all Packer-created build-time QEMU additional arguments
	buildTimeArgs, _ := state.Get("buildTimeQemuArgs").([]string)
	if len(buildTimeArgs) > 0 {
		ui.Say(fmt.Sprintf(
			"Removing %d build-time QEMU additional argument(s)", len(buildTimeArgs)))
		removeQemuArgsCommand := []string{
			"remove_qemu_additional_args.applescript", vmId,
			"--args",
		}
		removeQemuArgsCommand = append(removeQemuArgsCommand, buildTimeArgs...)
		_, err := driver.ExecuteOsaScript(removeQemuArgsCommand...)
		if err != nil {
			err := fmt.Errorf("error removing build-time QEMU additional arguments: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Get the absolute path of the output directory
	absOutputDir, err := filepath.Abs(s.OutputDir)
	if err != nil {
		err := fmt.Errorf("error getting absolute path of output directory: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Export via applescript POSIX only works with absolute paths.
	outputPath := filepath.Join(absOutputDir, s.OutputFilename+"."+s.Format)
	ui.Say("Exporting virtual machine...")

	// Export the VM to an UTM file
	if err := driver.Export(vmId, outputPath); err != nil {
		err := fmt.Errorf("error exporting VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// We set export path as the output directory with UTM file.
	// So it can be used as an artifact in the next steps.
	state.Put("exportPath", outputPath)

	return multistep.ActionContinue
}

func (s *StepExport) Cleanup(state multistep.StateBag) {}
