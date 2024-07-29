// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"errors"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type ExportConfig struct {
	// Only UTM, this specifies the output format
	// of the exported virtual machine. This defaults to utm.
	Format string `mapstructure:"format" required:"false"`
}

func (c *ExportConfig) Prepare(ctx *interpolate.Context) []error {
	if c.Format == "" {
		c.Format = "utm"
	}

	var errs []error
	if c.Format != "utm" {
		errs = append(errs,
			errors.New("invalid format, only 'utm' is allowed"))
	}

	return errs
}
