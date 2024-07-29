// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step uploads a file containing the UTM version, which
// can be useful for various provisioning reasons.
type StepUploadVersion struct {
	Path string
}

func (s *StepUploadVersion) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	comm := state.Get("communicator").(packersdk.Communicator)
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)

	if s.Path == "" {
		log.Println("UtmVersionFile is empty. Not uploading.")
		return multistep.ActionContinue
	}

	version, err := driver.Version()
	if err != nil {
		state.Put("error", fmt.Errorf("Error reading version for metadata upload: %s", err))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Uploading UTM version info (%s)", version))
	var data bytes.Buffer
	data.WriteString(version)
	if err := comm.Upload(s.Path, &data, nil); err != nil {
		state.Put("error", fmt.Errorf("Error uploading UTM version: %s", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *StepUploadVersion) Cleanup(state multistep.StateBag) {}
