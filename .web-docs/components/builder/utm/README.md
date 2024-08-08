Type: `utm-utm`
Artifact BuilderId: `naveenrajm7.utm`

The UTM Packer builder is able to create
[UTM](https://mac.getutm.app/) virtual machines and export them in
the .utm format., starting from an existing utm file (exported virtual machine
image).

The builder builds a virtual machine by importing an existing utm file.
It then boots this image, runs provisioners on this new VM, and exports that VM
to create the image. The imported machine is deleted prior to finishing the
build.

<!--
  A basic example on the usage of the builder. Multiple examples
  can be provided to highlight various build configurations.
-->
### Basic Example

Here is a basic example. This example is functional if you have an UTM matching
the settings here.

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
}
```

It is important to add a `shutdown_command`. By default Packer halts the virtual
machine and the file system may not be sync'd. Thus, changes made in a
provisioner might not be saved.

<!-- Builder Configuration Fields -->
## Configuration Reference

There are many configuration options available for the builder.

### Required:

<!-- Code generated from the comments of the Config struct in builder/utm/utm/config.go; DO NOT EDIT MANUALLY -->

- `checksum` (string) - The checksum for the source_path file. The type of the checksum is
  specified within the checksum field as a prefix, ex: "md5:{$checksum}".
  The type of the checksum can also be omitted and Packer will try to
  infer it based on string length. Valid values are "none", "{$checksum}",
  "md5:{$checksum}", "sha1:{$checksum}", "sha256:{$checksum}",
  "sha512:{$checksum}" or "file:{$path}". Here is a list of valid checksum
  values:
   * md5:090992ba9fd140077b0661cb75f7ce13
   * 090992ba9fd140077b0661cb75f7ce13
   * sha1:ebfb681885ddf1234c18094a45bbeafd91467911
   * ebfb681885ddf1234c18094a45bbeafd91467911
   * sha256:ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
   * ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
   * file:http://releases.ubuntu.com/20.04/SHA256SUMS
   * file:file://./local/path/file.sum
   * file:./local/path/file.sum
   * none
  Although the checksum will not be verified when it is set to "none",
  this is not recommended since these files can be very large and
  corruption does happen from time to time.

- `source_path` (string) - The filepath or URL to a UTM file that acts as the
  source of this build.

<!-- End of code generated from the comments of the Config struct in builder/utm/utm/config.go; -->



<!--
  Optional Configuration Fields

  Configuration options that are not required or have reasonable defaults
  should be listed under the optionals section. Defaults values should be
  noted in the description of the field
-->

#### Optional:

<!-- Code generated from the comments of the Config struct in builder/utm/utm/config.go; DO NOT EDIT MANUALLY -->

- `target_path` (string) - The path where the UTM file should be saved
  after download. By default, it will go in the packer cache, with a hash of
  the original filename as its name.

- `vm_name` (string) - This is the name of the UTM file for the new virtual machine, without
  the file extension. Make sure VMName in UTM after import is same
  as the UTM file name, By default this is packer-BUILDNAME,
  where "BUILDNAME" is the name of the build.

- `keep_registered` (bool) - Set this to true if you would like to keep
  the VM registered with UTM. Defaults to false.

- `skip_export` (bool) - Defaults to false. When enabled, Packer will
  not export the VM. Useful if the build output is not the resultant image,
  but created inside the VM.

<!-- End of code generated from the comments of the Config struct in builder/utm/utm/config.go; -->


<!-- Code generated from the comments of the UtmVersionConfig struct in builder/utm/common/utm_version_config.go; DO NOT EDIT MANUALLY -->

- `utm_version_file` (\*string) - The path within the virtual machine to
  upload a file that contains the UTM version that was used to create
  the machine. This information can be useful for provisioning. By default
  this is .utm_version, which will generally be upload it into the
  home directory. Set to an empty string to skip uploading this file, which
  can be useful when using the none communicator.

<!-- End of code generated from the comments of the UtmVersionConfig struct in builder/utm/common/utm_version_config.go; -->



### Export configuration

#### Optional:

<!-- Code generated from the comments of the ExportConfig struct in builder/utm/common/export_config.go; DO NOT EDIT MANUALLY -->

- `format` (string) - Only UTM, this specifies the output format
  of the exported virtual machine. This defaults to utm.

<!-- End of code generated from the comments of the ExportConfig struct in builder/utm/common/export_config.go; -->


### Shutdown configuration

#### Optional:

<!-- Code generated from the comments of the ShutdownConfig struct in builder/utm/common/shutdown_config.go; DO NOT EDIT MANUALLY -->

- `shutdown_command` (string) - The command to use to gracefully shut down the
  machine once all the provisioning is done. By default this is an empty
  string, which tells Packer to just forcefully shut down the machine unless a
  shutdown command takes place inside script so this may safely be omitted. If
  one or more scripts require a reboot it is suggested to leave this blank
  since reboots may fail and specify the final shutdown command in your
  last script.

- `shutdown_timeout` (duration string | ex: "1h5m2s") - The amount of time to wait after executing the
  shutdown_command for the virtual machine to actually shut down. If it
  doesn't shut down in this time, it is an error. By default, the timeout is
  5m or five minutes.

- `post_shutdown_delay` (duration string | ex: "1h5m2s") - The amount of time to wait after shutting
  down the virtual machine. If you get the error
  Error removing floppy controller, you might need to set this to 5m
  or so. By default, the delay is 0s or disabled.

- `disable_shutdown` (bool) - Packer normally halts the virtual machine after all provisioners have
  run when no `shutdown_command` is defined.  If this is set to `true`, Packer
  *will not* halt the virtual machine but will assume that you will send the stop
  signal yourself through the preseed.cfg or your final provisioner.
  Packer will wait for a default of 5 minutes until the virtual machine is shutdown.
  The timeout can be changed using `shutdown_timeout` option.

<!-- End of code generated from the comments of the ShutdownConfig struct in builder/utm/common/shutdown_config.go; -->


### Communicator configuration

#### Optional common fields:

<!-- Code generated from the comments of the Config struct in communicator/config.go; DO NOT EDIT MANUALLY -->

- `communicator` (string) - Packer currently supports three kinds of communicators:
  
  -   `none` - No communicator will be used. If this is set, most
      provisioners also can't be used.
  
  -   `ssh` - An SSH connection will be established to the machine. This
      is usually the default.
  
  -   `winrm` - A WinRM connection will be established.
  
  In addition to the above, some builders have custom communicators they
  can use. For example, the Docker builder has a "docker" communicator
  that uses `docker exec` and `docker cp` to execute scripts and copy
  files.

- `pause_before_connecting` (duration string | ex: "1h5m2s") - We recommend that you enable SSH or WinRM as the very last step in your
  guest's bootstrap script, but sometimes you may have a race condition
  where you need Packer to wait before attempting to connect to your
  guest.
  
  If you end up in this situation, you can use the template option
  `pause_before_connecting`. By default, there is no pause. For example if
  you set `pause_before_connecting` to `10m` Packer will check whether it
  can connect, as normal. But once a connection attempt is successful, it
  will disconnect and then wait 10 minutes before connecting to the guest
  and beginning provisioning.

<!-- End of code generated from the comments of the Config struct in communicator/config.go; -->


<!-- Code generated from the comments of the CommConfig struct in builder/utm/common/comm_config.go; DO NOT EDIT MANUALLY -->

- `host_port_min` (int) - The minimum port to use for the Communicator port on the host machine which is forwarded
  to the SSH or WinRM port on the guest machine. By default this is 2222.

- `host_port_max` (int) - The maximum port to use for the Communicator port on the host machine which is forwarded
  to the SSH or WinRM port on the guest machine. Because Packer often runs in parallel,
  Packer will choose a randomly available port in this range to use as the
  host port. By default this is 4444.

- `skip_nat_mapping` (bool) - Defaults to false. When enabled, Packer
  does not setup forwarded port mapping for communicator (SSH or WinRM) requests and uses ssh_port or winrm_port
  on the host to communicate to the virtual machine.

<!-- End of code generated from the comments of the CommConfig struct in builder/utm/common/comm_config.go; -->
