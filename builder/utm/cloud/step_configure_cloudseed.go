package cloud

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

// This step configures the VM to send cloud init seed file.
//
// Uses:
//
//	config *config
//	ui     packersdk.Ui
type stepConfigureCloudSeed struct {
	useCd               bool
	diskUnmountCommands map[string][]string
}

func (s *stepConfigureCloudSeed) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmId := state.Get("vmId").(string)

	if s.useCd {
		return s.attachCloudInitISO(ctx, state, driver, ui, vmId)
	} else {
		return s.configureCloudInitHTTP(ctx, state, driver, ui, vmId)
	}
}

func (s *stepConfigureCloudSeed) attachCloudInitISO(ctx context.Context, state multistep.StateBag, driver utmcommon.Driver, ui packersdk.Ui, vmId string) multistep.StepAction {
	ui.Say("Attaching cloud init seed ISO...")

	// initialize the diskUnmountCommands map
	s.diskUnmountCommands = map[string][]string{}
	var cdFilesPath string
	// Determine if we even have a cd_files disk to attach
	if cdPathRaw, ok := state.GetOk("cd_path"); ok {
		cdFilesPath = cdPathRaw.(string)
	} else {
		err := fmt.Errorf("no cd_files path found")
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Attaching cloud init seed ISO as a drive...")
	// Convert controllerName to the corresponding enum code
	controllerEnumCode, err := utmcommon.GetControllerEnumCode("virtio")
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	attachIsoCommand := []string{
		"attach_iso.applescript", vmId,
		"--interface", controllerEnumCode, // Working interface, replace as needed
		"--source", cdFilesPath,
		"--removable", "false", // Not removable, required for cloud init seed on virtio
	}
	output, err := driver.ExecuteOsaScript(attachIsoCommand...)
	if err != nil {
		err := fmt.Errorf("error attaching cloud init seed ISO: %s", err)
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
		s.diskUnmountCommands["cloud_seed"] = unmountCommand
	} else {
		err := fmt.Errorf("error extracting UUID from output: %s", output)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("disk_unmount_commands", s.diskUnmountCommands)

	return multistep.ActionContinue
}

func (s *stepConfigureCloudSeed) configureCloudInitHTTP(ctx context.Context, state multistep.StateBag, driver utmcommon.Driver, ui packersdk.Ui, vmId string) multistep.StepAction {
	// Get the host IP and HTTP port
	hostIP := state.Get("http_ip").(string)
	// If VM does not have an IP for emulated VLAN, we need to use the gateway IP of vmnet
	// TODO: Get Host IP automatically from Mac's Shared_Network_Address,
	// Ex: "192.168.x.x"
	httpPort := state.Get("http_port").(int)

	// Add Qemu args to send cloud init seed file
	ui.Say("Configuring VM to send cloud init seed file...")
	cloudQemuArg := fmt.Sprintf("-smbios type=1,serial=ds=nocloud-net;seedfrom=http://%s:%d/", hostIP, httpPort)

	// Collect all args: user qemuargs (already set) + cloud-init arg
	// IMPORTANT: The AppleScript replaces (not appends) all QEMU additional args,
	// so we must re-include any previously-set user args.
	addQemuArgsCommand := []string{
		"add_qemu_additional_args.applescript", vmId,
		"--args",
	}
	if userArgs, ok := state.Get("userQemuArgs").([]string); ok {
		addQemuArgsCommand = append(addQemuArgsCommand, userArgs...)
	}
	addQemuArgsCommand = append(addQemuArgsCommand, cloudQemuArg)

	ui.Say("Adding QEMU additional arguments...")
	_, err := driver.ExecuteOsaScript(addQemuArgsCommand...)
	if err != nil {
		err := fmt.Errorf("error adding QEMU additional arguments: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	// Append the Cloud QEMU argument to build-time args for later cleanup
	buildTimeArgs, _ := state.Get("buildTimeQemuArgs").([]string)
	buildTimeArgs = append(buildTimeArgs, cloudQemuArg)
	state.Put("buildTimeQemuArgs", buildTimeArgs)

	return multistep.ActionContinue
}

func (s *stepConfigureCloudSeed) Cleanup(state multistep.StateBag) {
	if len(s.diskUnmountCommands) == 0 {
		return
	}

	driver := state.Get("driver").(utmcommon.Driver)
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
