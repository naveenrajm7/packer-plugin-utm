// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepConfigureQemuArgs adds user-specified QEMU additional arguments to the VM.
// These args persist in the exported VM (they are intentional configuration).
//
// Uses:
//
//	driver Driver
//	ui     packersdk.Ui
//	vmId   string
type StepConfigureQemuArgs struct {
	QemuArgs [][]string
}

func (s *StepConfigureQemuArgs) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if len(s.QemuArgs) == 0 {
		log.Println("[INFO] No user QEMU args to configure, skipping...")
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmId := state.Get("vmId").(string)

	// Join each inner []string into a single QEMU arg string
	var qemuArgStrings []string
	for _, args := range s.QemuArgs {
		qemuArgStrings = append(qemuArgStrings, strings.Join(args, " "))
	}

	ui.Say(fmt.Sprintf("Adding %d user QEMU additional argument(s)...", len(qemuArgStrings)))
	for _, arg := range qemuArgStrings {
		log.Printf("[INFO] QEMU arg: %s", arg)
	}

	addQemuArgsCommand := []string{
		"add_qemu_additional_args.applescript", vmId,
		"--args",
	}
	addQemuArgsCommand = append(addQemuArgsCommand, qemuArgStrings...)

	_, err := driver.ExecuteOsaScript(addQemuArgsCommand...)
	if err != nil {
		err := fmt.Errorf("error adding user QEMU additional arguments: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Store user args in state bag so later steps (VNC, cloud-init) can
	// re-include them when calling add_qemu_additional_args (which replaces, not appends).
	state.Put("userQemuArgs", qemuArgStrings)

	return multistep.ActionContinue
}

func (s *StepConfigureQemuArgs) Cleanup(state multistep.StateBag) {
	// No cleanup needed: user-specified QEMU args should persist in the exported VM.
}
