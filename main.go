// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-scaffolding/builder/scaffolding"
	scaffoldingData "github.com/hashicorp/packer-plugin-scaffolding/datasource/scaffolding"
	scaffoldingPP "github.com/hashicorp/packer-plugin-scaffolding/post-processor/scaffolding"
	scaffoldingProv "github.com/hashicorp/packer-plugin-scaffolding/provisioner/scaffolding"
	scaffoldingVersion "github.com/hashicorp/packer-plugin-scaffolding/version"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("my-builder", new(scaffolding.Builder))
	pps.RegisterProvisioner("my-provisioner", new(scaffoldingProv.Provisioner))
	pps.RegisterPostProcessor("my-post-processor", new(scaffoldingPP.PostProcessor))
	pps.RegisterDatasource("my-datasource", new(scaffoldingData.Datasource))
	pps.SetVersion(scaffoldingVersion.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
