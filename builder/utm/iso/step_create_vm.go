package iso

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

// This step creates the actual virtual machine.
//
// Produces:
//
//	vmName string - The name of the VM
type stepCreateVM struct {
	vmName string
}

func (s *stepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)

	name := config.VMName

	commands := [][]string{}
	commands = append(commands, []string{
		"create_vm.applescript", "--name", name,
		"--backend", config.VMBackend,
		"--arch", config.VMArch,
	})

	// customize
	commands = append(commands, []string{
		"customize_vm.applescript", name,
		"--cpus", strconv.Itoa(config.HWConfig.CpuCount),
		"--memory", strconv.Itoa(config.HWConfig.MemorySize),
	})

	ui.Say("Creating virtual machine...")
	for _, command := range commands {
		_, err := driver.ExecuteOsaScript(command...)
		if err != nil {
			err := fmt.Errorf("error creating VM: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Set the VM name property on the first command
		if s.vmName == "" {
			s.vmName = name
		}
	}

	// Set the final name in the state bag so others can use it
	state.Put("vmName", s.vmName)

	return multistep.ActionContinue
}

func (s *stepCreateVM) Cleanup(state multistep.StateBag) {
	if s.vmName == "" {
		return
	}

	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if (config.KeepRegistered) && (!cancelled && !halted) {
		ui.Say("Keeping virtual machine registered with VirtualBox host (keep_registered = true)")
		return
	}

	ui.Say("Deregistering and deleting VM...")
	if err := driver.Delete(s.vmName); err != nil {
		ui.Error(fmt.Sprintf("Error deleting VM: %s", err))
	}
}
