// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestQemuConfigPrepare_empty(t *testing.T) {
	c := new(QemuConfig)
	errs := c.Prepare(nil)
	if len(errs) > 0 {
		t.Fatalf("should not have errors: %#v", errs)
	}
}

func TestQemuConfigPrepare_validArgs(t *testing.T) {
	c := &QemuConfig{
		QemuArgs: [][]string{
			{"-accel", "hvf"},
			{"-cpu", "host"},
		},
	}
	errs := c.Prepare(nil)
	if len(errs) > 0 {
		t.Fatalf("should not have errors: %#v", errs)
	}
}

func TestQemuConfigPrepare_emptyInnerArray(t *testing.T) {
	c := &QemuConfig{
		QemuArgs: [][]string{
			{},
		},
	}
	errs := c.Prepare(nil)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got: %#v", errs)
	}
}

func TestQemuConfigPrepare_whitespaceOnly(t *testing.T) {
	c := &QemuConfig{
		QemuArgs: [][]string{
			{" ", "  "},
		},
	}
	errs := c.Prepare(nil)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got: %#v", errs)
	}
}
