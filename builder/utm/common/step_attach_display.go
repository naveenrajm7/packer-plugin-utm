package common

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step attaches a display to the virtual machine.
type StepAttachDisplay struct {
	HardwareType          string
	detachDisplayCommands [][]string
}

func (s *StepAttachDisplay) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(Driver)
	vmId := state.Get("vmId").(string)

	// Check if HardwareType is empty
	if s.HardwareType == "" {
		ui.Say("No hardware type specified for display. Skipping display attachment.")
		return multistep.ActionContinue
	}

	ui.Say(fmt.Sprintf("Attaching display with hardware type '%s'...", s.HardwareType))

	// Attach the display
	command := []string{
		"add_qemu_display.applescript", vmId,
		"--hardware", s.HardwareType,
	}

	_, err := driver.ExecuteOsaScript(command...)
	if err != nil {
		err := fmt.Errorf("error attaching display: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Build the detach command
	detachCommand := []string{
		"remove_qemu_display_by_name.applescript", vmId,
		"--hardware", s.HardwareType,
	}
	s.detachDisplayCommands = append(s.detachDisplayCommands, detachCommand)

	// Save the detach commands in the state for cleanup
	state.Put("detach_display_commands", s.detachDisplayCommands)

	ui.Say(fmt.Sprintf("Display with hardware type '%s' attached successfully.", s.HardwareType))
	return multistep.ActionContinue
}

func (s *StepAttachDisplay) Cleanup(state multistep.StateBag) {
	if len(s.detachDisplayCommands) == 0 {
		return
	}

	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(Driver)

	// Check if the detach command is already
	// executed in the previous step
	_, ok := state.GetOk("detached_displays")

	if !ok {
		ui.Say("Detaching displays...")
		for _, command := range s.detachDisplayCommands {
			_, err := driver.ExecuteOsaScript(command...)
			if err != nil {
				log.Printf("error detaching display: %s", err)
			}
		}
	}
}
