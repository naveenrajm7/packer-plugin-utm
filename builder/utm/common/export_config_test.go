// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

func TestExportConfigPrepare_BootWait(t *testing.T) {
	var c *ExportConfig
	var errs []error

	// Bad
	c = new(ExportConfig)
	c.Format = "illegal"
	errs = c.Prepare(interpolate.NewContext())
	if len(errs) == 0 {
		t.Fatalf("bad: %#v", errs)
	}

	// Good
	c = new(ExportConfig)
	c.Format = "utm"
	errs = c.Prepare(interpolate.NewContext())
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}

	// Good
	c = new(ExportConfig)
	c.Format = "utm"
	errs = c.Prepare(interpolate.NewContext())
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}
}

// TODO: add export opts test, when utm export with options is supported
