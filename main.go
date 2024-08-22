// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/plugin"

	"github.com/naveenrajm7/packer-plugin-utm/builder/utm/iso"
	"github.com/naveenrajm7/packer-plugin-utm/builder/utm/utm"
	utmPPzip "github.com/naveenrajm7/packer-plugin-utm/post-processor/zip"
	"github.com/naveenrajm7/packer-plugin-utm/version"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("iso", new(iso.Builder))
	pps.RegisterBuilder("utm", new(utm.Builder))
	pps.RegisterPostProcessor("zip", new(utmPPzip.PostProcessor))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
