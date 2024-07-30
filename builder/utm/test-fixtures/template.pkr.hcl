# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

source "utm-utm" "basic-example" {
  mock = "mock-config"
}

build {
  sources = [
    "source.utm-utm.basic-example"
  ]

  provisioner "shell-local" {
    inline = [
      "echo build generated data: ${build.GeneratedMockData}",
    ]
  }
}
