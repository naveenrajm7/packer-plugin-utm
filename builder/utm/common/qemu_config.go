// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"fmt"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type QemuConfig struct {
	// Arbitrary QEMU arguments that are passed to the UTM virtual machine
	// as QEMU additional arguments. Each element is a list of strings
	// that are joined with a space to form a single QEMU argument.
	// These arguments persist in the exported VM.
	//
	// Usage example:
	//
	// In JSON:
	// ```json
	// "qemuargs": [
	//   ["-accel", "hvf"],
	//   ["-cpu", "host"]
	// ]
	// ```
	//
	// In HCL2:
	// ```hcl
	// qemuargs = [
	//   ["-accel", "hvf"],
	//   ["-cpu", "host"],
	// ]
	// ```
	QemuArgs [][]string `mapstructure:"qemuargs" required:"false"`
}

func (c *QemuConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	for i, args := range c.QemuArgs {
		if len(args) == 0 {
			errs = append(errs, fmt.Errorf("qemuargs[%d]: empty argument list", i))
			continue
		}
		joined := strings.Join(args, " ")
		if strings.TrimSpace(joined) == "" {
			errs = append(errs, fmt.Errorf("qemuargs[%d]: argument resolves to empty string", i))
		}
	}

	return errs
}
