// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type UtmBundleConfig struct {
	// Defaults to false. When enabled, Packer includes
	// any attached ISO disc devices into the final virtual machine. Useful for
	// some live distributions that require installation media to continue to be
	// attached after installation.
	BundleISO bool `mapstructure:"bundle_iso" required:"false"`
}

func (c *UtmBundleConfig) Prepare(ctx *interpolate.Context) []error {
	return nil
}
