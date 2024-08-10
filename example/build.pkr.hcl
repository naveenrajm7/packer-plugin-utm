# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

packer {
  required_plugins {
    utm = {
      version = ">=v0.0.2"
      source  = "github.com/naveenrajm7/utm"
    }
  }
}

source "utm-utm" "basic-example" {
  source_path = "source.utm"
  vm_name = "source"
  ssh_username = "packer"
  ssh_password = "packer"
  shutdown_command = "echo 'packer' | sudo -S shutdown -P now"
}

build {
  sources = [ "source.utm-utm.basic-example" ]

  post-processor "utm-zip" {
    output = "{{.BuildName}}_vagrant_utm.zip"
    keep_input_artifact = true
  }
}
