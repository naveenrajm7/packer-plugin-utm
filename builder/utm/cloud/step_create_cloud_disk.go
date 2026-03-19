package cloud

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/tmp"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

// This step creates the virtual disk from cloud image that will be used as the
// hard drive for the virtual machine.
type stepCreateCloudDisk struct {
	ResizedCloudImagePath string
}

func (s *stepCreateCloudDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmId := state.Get("vmId").(string)

	cloudImagePath := state.Get("iso_path").(string)

	// Create main disk seperately for cloud image, since it uses source file
	// Additional disks are created with use of size

	// Create a temporary file to be our cloud image drive
	TMPF, err := tmp.File("packer*.iso")
	// Set the path so we can remove it later
	TMPPath := TMPF.Name()
	_ = TMPF.Close()
	_ = os.Remove(TMPPath)
	if err != nil {
		state.Put("error",
			fmt.Errorf("error creating temporary file for Cloud image: %s", err))
		return multistep.ActionHalt
	}
	log.Printf("Temp cloud image path: %s", TMPPath)
	s.ResizedCloudImagePath = TMPPath
	// Create a copy of the original cloud image
	ui.Say("Creating a copy of the original cloud image...")

	err = copyFile(cloudImagePath, s.ResizedCloudImagePath)
	if err != nil {
		err := fmt.Errorf("error copying cloud image: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// if ResizeCloudImage is true, resize the cloud image
	// Use qemu-img to increase the size of the cloud image to config.DiskSize
	// This is required as default disk size of cloud image is small
	// and we need to honor the user provided disk size
	if config.ResizeCloudImage {
		// Resize the cloud image using qemu-img
		ui.Say(fmt.Sprintf("Resizing cloud image with size %d MiB...", config.DiskSize))
		diskSizeStr := fmt.Sprintf("%dM", config.DiskSize)
		cmd := exec.Command("qemu-img", "resize", s.ResizedCloudImagePath, diskSizeStr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			err := fmt.Errorf("error resizing cloud image: %s, output: %s", err, string(output))
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		ui.Say("Cloud image resized successfully.")
	}

	// Convert controllerName to the corresponding enum code
	controllerEnumCode, err := utmcommon.GetControllerEnumCode(config.HardDriveInterface)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	// Attach the cloud image as a non removable drive
	command := []string{
		"attach_iso.applescript", vmId,
		"--interface", controllerEnumCode,
		"--source", s.ResizedCloudImagePath,
		"--removable", "false",
	}

	_, err = driver.ExecuteOsaScript(command...)
	if err != nil {
		err := fmt.Errorf("error creating hard drive: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Create additional disks
	// We do not give names to the disks, as UTM does not support it
	diskSizes := []uint{}
	if len(config.AdditionalDiskSize) > 0 {
		diskSizes = append(diskSizes, config.AdditionalDiskSize...)
	}

	// Create all required disks
	for i := range diskSizes {
		ui.Say(fmt.Sprintf("Creating hard drive with size %d MiB...", diskSizes[i]))

		// Convert controllerName to the corresponding enum code
		controllerEnumCode, err := utmcommon.GetControllerEnumCode(config.HardDriveInterface)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		command := []string{
			"add_drive.applescript", vmId,
			"--interface", controllerEnumCode,
			"--size", strconv.FormatUint(uint64(diskSizes[i]), 10),
		}
		_, err = driver.ExecuteOsaScript(command...)
		if err != nil {
			err := fmt.Errorf("error creating hard drive: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// In UTM disk creation and attaching are done in the same step

	return multistep.ActionContinue
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destinationFile.Close() }()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func (s *stepCreateCloudDisk) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	ui.Say("Cleaning up copied and resized cloud image...")
	err := os.Remove(s.ResizedCloudImagePath)
	if err != nil {
		ui.Error(fmt.Sprintf("error removing copied and resized cloud image: %s", err))
	}
}
