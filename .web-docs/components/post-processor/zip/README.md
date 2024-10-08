Artifact BuilderId: `naveenrajm7.utm.post-processor.zip`

<!--
  Include a short description about the post-processor. This is a good place
  to call out what the post-processor does, and any additional text that might
  be helpful to a user. See https://www.packer.io/docs/provisioner/null
-->

The Packer UTM zip post-processor takes an artifact with .utm directory 
 and compresses the artifact into a single zip archive.

<!--
  A basic example on the usage of the post-processor. Multiple examples
  can be provided to highlight various configurations.

-->
## Basic Example


```hcl
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
```

<!-- Post-Processor Configuration Fields -->
## Configuration Reference



<!--
  Optional Configuration Fields

  Configuration options that are not required or have reasonable defaults
  should be listed under the optionals section. Defaults values should be
  noted in the description of the field
-->

### Optional:

<!-- Code generated from the comments of the Config struct in post-processor/zip/post-processor.go; DO NOT EDIT MANUALLY -->

- `output` (string) - Fields from config file

<!-- End of code generated from the comments of the Config struct in post-processor/zip/post-processor.go; -->
