package common

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step attaches the boot ISO, to the
// virtual machine, if present.
type StepAttachISOs struct {
	AttachBootISO bool
	ISOInterface  string
	// diskUnmountCommands map[string][]string
}

func (s *StepAttachISOs) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	// Check whether there is anything to attach
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Mounting ISOs...")
	diskMountMap := map[string]string{}
	// Track the bootable iso (only used in utm-iso builder. )
	if s.AttachBootISO {
		isoPath := state.Get("iso_path").(string)
		diskMountMap["boot_iso"] = isoPath
	}

	if len(diskMountMap) == 0 {
		ui.Message("No ISOs to mount; continuing...")
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(Driver)
	vmName := state.Get("vmName").(string)

	for diskCategory, isoPath := range diskMountMap {
		// If it's a symlink, resolve it to its target.
		resolvedIsoPath, err := filepath.EvalSymlinks(isoPath)
		if err != nil {
			err := fmt.Errorf("error resolving symlink for ISO: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		isoPath = resolvedIsoPath

		// We may have different potential iso we can attach.
		switch diskCategory {
		case "boot_iso":
			ui.Message("Mounting boot ISO...")
		}

		// Attach the ISO
		command := []string{
			"attach_iso.applescript", vmName,
			"--iso", isoPath,
		}
		if _, err := driver.ExecuteOsaScript(command...); err != nil {
			err := fmt.Errorf("error attaching ISO: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}
	return multistep.ActionContinue
}

func (s *StepAttachISOs) Cleanup(state multistep.StateBag) {}
