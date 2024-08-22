// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step pauses the build process and asks the user to confirm
// if the necessary steps are done.
// Just a temporary step to pause the build process.

type StepPause struct{}

func (s *StepPause) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)

	// ask user to confirm if the necessary steps are done
	confirmOption, err := ui.Ask("confirm you have done the necessary [Y/n]:")

	if err != nil {
		err := fmt.Errorf("error during export step: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if confirmOption == "Y" || confirmOption == "y" {
		// Proceed with the next steps
		ui.Say("Proceeding assuming necessary manual steps are done...")

		return multistep.ActionContinue
	} else {
		ui.Say("Build halted by user.")
		return multistep.ActionHalt
	}

}

func (s *StepPause) Cleanup(state multistep.StateBag) {}
