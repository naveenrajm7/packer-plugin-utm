// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"reflect"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

func TestVBoxBundleConfigPrepare_VBoxBundle(t *testing.T) {
	// Test with empty
	c := new(UtmBundleConfig)
	errs := c.Prepare(interpolate.NewContext())
	if len(errs) > 0 {
		t.Fatalf("err: %#v", errs)
	}

	if !reflect.DeepEqual(*c, UtmBundleConfig{BundleISO: false}) {
		t.Fatalf("bad: %#v", c)
	}

	// Test with a good one
	c = new(UtmBundleConfig)
	c.BundleISO = true
	errs = c.Prepare(interpolate.NewContext())
	if len(errs) > 0 {
		t.Fatalf("err: %#v", errs)
	}

	expected := UtmBundleConfig{
		BundleISO: true,
	}

	if !reflect.DeepEqual(*c, expected) {
		t.Fatalf("bad: %#v", c)
	}
}
