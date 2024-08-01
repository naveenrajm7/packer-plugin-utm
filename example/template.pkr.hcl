# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0
packer {
  required_plugins {
    scaffolding = {
      version = ">=v0.0.1"
      source  = "github.com/naveenrajm7/utm"
    }
  }
}

source "utm-utm" "vagrant-example" {
  source_path = "/Users/naveenrajm/Developer/UTMvagrant/utm_gallery/Debian11G.utm"
  vm_name = "Debian11G"
  ssh_username = "debian"
  ssh_password = "debian"
  keep_registered = true
}

build {
  sources = [
    "source.utm-utm.vagrant-example"
  ]

  provisioner "shell-local" {
    inline = [
      "echo Success!",
    ]
  }

  // compress does not compress directories, only files
  // but our output UTM VM is a directory, so we need to
  // compress it into a zip file using custom post-processor
  post-processor "compress" {
    output = "{{.BuildName}}_utm.zip"
    keep_input_artifact = true
  }
}
