package common

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step attaches the boot ISO, cd_files iso, and guest additions to the
// virtual machine, if present.
//
// ISOs are attached in a specific order to ensure predictable drive letter
// assignment in Windows guests:
//  1. boot_iso - The installation ISO (typically C: after install, but mounted first)
//  2. cd_files - User-provided files ISO (typically D: in Windows)
//  3. guest_additions - UTM guest tools ISO (typically E: in Windows)
//
// This ordering is critical for Windows installations where scripts may depend
// on knowing which drive letter to use for accessing files or running installers.
type StepAttachISOs struct {
	AttachBootISO           bool
	ISOInterface            string
	GuestAdditionsMode      string
	GuestAdditionsInterface string
	diskUnmountCommands     map[string][]string
}

// diskToMount represents an ISO to mount with its category and path
type diskToMount struct {
	category string
	isoPath  string
}

func (s *StepAttachISOs) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	// Check whether there is anything to attach
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Mounting ISOs...")
	// Use a slice to maintain predictable order for consistent drive letters in Windows
	disksToMount := []diskToMount{}
	s.diskUnmountCommands = map[string][]string{}

	// Track the bootable iso (only used in utm-iso builder. )
	// Boot ISO should be first for predictable drive letters
	if s.AttachBootISO {
		isoPath := state.Get("iso_path").(string)
		// Convert to absolute path if it's not already
		if !filepath.IsAbs(isoPath) {
			absPath, err := filepath.Abs(isoPath)
			if err != nil {
				err := fmt.Errorf("error converting iso_path to absolute path: %s", err)
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
			isoPath = absPath
		}
		disksToMount = append(disksToMount, diskToMount{
			category: "boot_iso",
			isoPath:  isoPath,
		})
	}

	// Determine if we even have a cd_files disk to attach
	// cd_files should be second for predictable drive letters (usually D: in Windows)
	if cdPathRaw, ok := state.GetOk("cd_path"); ok {
		cdFilesPath := cdPathRaw.(string)
		// Convert to absolute path if it's not already
		if !filepath.IsAbs(cdFilesPath) {
			absPath, err := filepath.Abs(cdFilesPath)
			if err != nil {
				err := fmt.Errorf("error converting cd_path to absolute path: %s", err)
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
			cdFilesPath = absPath
		}
		disksToMount = append(disksToMount, diskToMount{
			category: "cd_files",
			isoPath:  cdFilesPath,
		})
	}

	// Determine if we have guest additions to attach
	// Guest additions should be last for predictable drive letters (usually E: in Windows)
	if s.GuestAdditionsMode != GuestAdditionsModeAttach {
		log.Println("Not attaching guest additions since we're uploading.")
	} else {
		// Get the guest additions path since we're doing it
		guestAdditionsPath := state.Get("guest_additions_path").(string)
		// Convert to absolute path if it's not already
		if !filepath.IsAbs(guestAdditionsPath) {
			absPath, err := filepath.Abs(guestAdditionsPath)
			if err != nil {
				err := fmt.Errorf("error converting guest_additions_path to absolute path: %s", err)
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
			guestAdditionsPath = absPath
		}
		disksToMount = append(disksToMount, diskToMount{
			category: "guest_additions",
			isoPath:  guestAdditionsPath,
		})
	}

	if len(disksToMount) == 0 {
		ui.Say("No ISOs to mount; continuing...")
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(Driver)
	vmId := state.Get("vmId").(string)

	// Iterate over the ISOs to attach in the specified order
	// This ensures predictable drive letter assignment in Windows guests
	for _, disk := range disksToMount {
		diskCategory := disk.category
		isoPath := disk.isoPath
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
		var controllerName string
		switch diskCategory {
		case "boot_iso":
			controllerName = s.ISOInterface
			ui.Say("Mounting boot ISO...")
		case "guest_additions":
			controllerName = s.GuestAdditionsInterface
			ui.Say("Mounting guest additions ISO...")
		case "cd_files":
			controllerName = "usb"
			ui.Say("Mounting cd_files ISO...")
		}

		// Convert controllerName to the corresponding enum code
		controllerEnumCode, err := GetControllerEnumCode(controllerName)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		// Attach the ISO
		command := []string{
			"attach_iso.applescript", vmId,
			"--interface", controllerEnumCode,
			"--source", isoPath,
		}

		output, err := driver.ExecuteOsaScript(command...)
		if err != nil {
			err := fmt.Errorf("error attaching ISO: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Track the disks we've mounted so we can remove them without having
		// to re-derive what was mounted where

		// Regular expression to capture the UUID from the output
		re := regexp.MustCompile(`[0-9a-fA-F-]{36}`)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 0 {
			uuid := matches[0] // Capture the UUID
			unmountCommand := []string{
				"remove_drive.applescript", vmId, uuid,
			}
			s.diskUnmountCommands[diskCategory] = unmountCommand
		} else {
			err := fmt.Errorf("error extracting UUID from output: %s", output)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	state.Put("disk_unmount_commands", s.diskUnmountCommands)
	return multistep.ActionContinue
}

func (s *StepAttachISOs) Cleanup(state multistep.StateBag) {
	if len(s.diskUnmountCommands) == 0 {
		return
	}

	driver := state.Get("driver").(Driver)
	_, ok := state.GetOk("detached_isos")

	if !ok {
		for _, command := range s.diskUnmountCommands {
			_, err := driver.ExecuteOsaScript(command...)
			if err != nil {
				log.Printf("error detaching iso: %s", err)
			}
		}
	}
}
