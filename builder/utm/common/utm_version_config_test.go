// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestUtmVersionConfigPrepare_BootWait(t *testing.T) {
	var c *UtmVersionConfig
	var errs []error

	// Test empty
	c = new(UtmVersionConfig)
	errs = c.Prepare("ssh")
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}

	if *c.UtmVersionFile != ".utm_version" {
		t.Fatalf("bad value: %s", *c.UtmVersionFile)
	}

	// Test with a good one
	c = new(UtmVersionConfig)
	filename := "foo"
	c.UtmVersionFile = &filename
	errs = c.Prepare("ssh")
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}

	if *c.UtmVersionFile != "foo" {
		t.Fatalf("bad value: %s", *c.UtmVersionFile)
	}
}

func TestUtmVersionConfigPrepare_empty(t *testing.T) {
	var c *UtmVersionConfig
	var errs []error

	// Test with nil value
	c = new(UtmVersionConfig)
	c.UtmVersionFile = nil
	errs = c.Prepare("ssh")
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}

	if *c.UtmVersionFile != ".utm_version" {
		t.Fatalf("bad value: %s", *c.UtmVersionFile)
	}

	// Test with empty name
	c = new(UtmVersionConfig)
	filename := ""
	c.UtmVersionFile = &filename
	errs = c.Prepare("ssh")
	if len(errs) > 0 {
		t.Fatalf("should not have error: %s", errs)
	}

	if *c.UtmVersionFile != "" {
		t.Fatalf("bad value: %s", *c.UtmVersionFile)
	}
}

func TestUtmVersionConfigPrepare_communicator(t *testing.T) {
	var c *UtmVersionConfig
	var errs []error

	// Test with 'none' communicator and non-empty utm_version_file
	c = new(UtmVersionConfig)
	filename := "test"
	c.UtmVersionFile = &filename
	errs = c.Prepare("none")
	if len(errs) == 0 {
		t.Fatalf("should have an error")
	}
}
