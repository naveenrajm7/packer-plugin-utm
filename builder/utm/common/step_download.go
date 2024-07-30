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

// SDK Version StepDownload does not handle the local UTM file
// So we need to modify the StepDownload to handle the local UTM file

// StepDownload downloads a remote file using the download client within
// this package. This step handles setting up the download configuration,
// progress reporting, interrupt handling, etc.
//
// Uses:
//
//	cache packer.Cache
//	ui    packersdk.Ui
type StepDownload struct {
	// The checksum and the type of the checksum for the download
	Checksum string

	// A short description of the type of download being done. Example:
	// "ISO" or "Guest Additions"
	Description string

	// The name of the key where the final path of the ISO will be put
	// into the state.
	ResultKey string

	// The path where the result should go, otherwise it goes to the
	// cache directory.
	TargetPath string

	// A list of URLs to attempt to download this thing.
	Url []string

	// Extension is the extension to force for the file that is downloaded.
	// Some systems require a certain extension. If this isn't set, the
	// extension on the URL is used. Otherwise, this will be forced
	// on the downloaded file for every URL.
	Extension string
}

func (s *StepDownload) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	// TODO: Change this to handle the local UTM file
	// by checking file existence
	// setting the file path to the ResultKey
	if len(s.Url) == 0 {
		log.Printf("No URLs were provided to Step Download. Continuing...")
		return multistep.ActionContinue
	}

	defer log.Printf("Leaving retrieve loop for %s", s.Description)

	ui := state.Get("ui").(packersdk.Ui)
	ui.Say(fmt.Sprintf("Retrieving %s", s.Description))

	var errs []error

	for _, source := range s.Url {
		if ctx.Err() != nil {
			state.Put("error", fmt.Errorf("download cancelled: %v", errs))
			return multistep.ActionHalt
		}
		ui.Say(fmt.Sprintf("Trying %s", source))
		var err error
		var dst string
		if s.Description == "UTM" && strings.HasSuffix(source, ".utm") {
			// TODO(adrien): make go-getter allow using files in place.
			// ovf files usually point to a file in the same directory, so
			// using them in place is the only way.
			ui.Say("Using utm file inplace")
			dst = source
		}
		if err == nil {
			state.Put(s.ResultKey, dst)
			// Track the URL you actually used for the download.
			state.Put("SourceImageURL", source)
			return multistep.ActionContinue
		}
		// may be another url will work
		errs = append(errs, err)
	}

	err := fmt.Errorf("error downloading %s: %v", s.Description, errs)
	state.Put("error", err)
	ui.Error(err.Error())
	return multistep.ActionHalt
}

func (s *StepDownload) Cleanup(multistep.StateBag) {}
