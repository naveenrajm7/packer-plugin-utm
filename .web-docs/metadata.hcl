# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Details on using this Integration template can be found at https://github.com/hashicorp/integration-template
# This metadata.hcl file and the adjacent `components` docs directory should
# be kept in a `.web-docs` directory at the root of your plugin repository.
integration {
  name = "Integration Template"
  description = "This is an integration template"
  identifier = "packer/hashicorp/scaffolding"
  flags = [
    # Remove if the plugin does not conform to the HCP Packer requirements.
    #
    # Please refer to our docs if you want your plugin to be compatible with
    # HCP Packer: https://developer.hashicorp.com/packer/docs/plugins/creation/hcp-support
    "hcp-ready",
    # This signals that the plugin is unmaintained and will eventually not be
    # working with a future version of Packer.
    #
    # On the integrations, this will end-up as an icon on the plugin's main card.
    "archived",
  ]
  docs {
    # If you'd prefer not to publish docs on HashiCorp websites, you can
    # set `process_docs` to `false`. If `process_docs` is `false`, you MUST
    # provide a `external_url` so we can link back to your plugin repo.
    process_docs = true
    # Note that the README location is relative to this file. We recommend
    # keeping the default value, as the adjacent `compile-to-webdocs` script
    # will automatically copy the README from the `docs` directory of this
    # repository to the correct location.
    readme_location = "./README.md"
    # `external_url` allows us to link back to your plugin repo.
    external_url = "https://github.com/hashicorp/integration-template"
  }
  license {
    type = "MPL-2.0"
    url = "https://github.com/hashicorp/integration-template/blob/main/LICENSE.md"
  }
  component {
    type = "builder"
    name = "Component Name (e.g HappyCloud EBS)"
    slug = "name"
  }
  component {
    type = "provisioner"
    name = "Component Name (e.g HappyCloud Shell)"
    slug = "name"
  }
  component {
    type = "post-processor"
    name = "Component Name"
    slug = "name"
  }
  component {
    type = "data-source"
    name = "Component Name"
    slug = "name"
  }
}
