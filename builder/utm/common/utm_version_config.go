// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"fmt"
)

type UtmVersionConfig struct {
	// The path within the virtual machine to
	// upload a file that contains the UTM version that was used to create
	// the machine. This information can be useful for provisioning. By default
	// this is .utm_version, which will generally be upload it into the
	// home directory. Set to an empty string to skip uploading this file, which
	// can be useful when using the none communicator.
	UtmVersionFile *string `mapstructure:"utm_version_file" required:"false"`
}

func (c *UtmVersionConfig) Prepare(communicatorType string) []error {
	var errs []error

	if c.UtmVersionFile == nil {
		default_file := ".utm_version"
		c.UtmVersionFile = &default_file
	}

	if communicatorType == "none" && *c.UtmVersionFile != "" {
		errs = append(errs, fmt.Errorf("utm_version_file has to be an "+
			"empty string when communicator = 'none'"))
	}

	return errs
}
