//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package iso

import (
	"errors"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

type Config struct {
	common.PackerConfig        `mapstructure:",squash"`
	commonsteps.HTTPConfig     `mapstructure:",squash"`
	commonsteps.ISOConfig      `mapstructure:",squash"`
	bootcommand.VNCConfig      `mapstructure:",squash"`
	utmcommon.ExportConfig     `mapstructure:",squash"`
	utmcommon.OutputConfig     `mapstructure:",squash"`
	utmcommon.ShutdownConfig   `mapstructure:",squash"`
	utmcommon.CommConfig       `mapstructure:",squash"`
	utmcommon.HWConfig         `mapstructure:",squash"`
	utmcommon.UtmVersionConfig `mapstructure:",squash"`
	utmcommon.UtmBundleConfig  `mapstructure:",squash"`

	// This is an array of tuples of boot commands, to type when the virtual
	// machine is booted. The first element of the tuple is the actual boot
	// command. The second element of the tuple, which is optional, is a
	// description of what the boot command does. This is intended to be used for
	// interactive installers that requires many commands to complete the
	// installation. Both the command and the description will be printed when
	// logging is enabled. When debug mode is enabled Packer will pause after
	// typing each boot command. This will make it easier to follow along the
	// installation process and make sure the Packer and the installer are in
	// sync. `boot_steps` and `boot_commands` are mutually exclusive.
	//
	// Example:
	//
	// In HCL:
	// ```hcl
	// boot_steps = [
	//   ["1<enter><wait5>", "Install NetBSD"],
	//   ["a<enter><wait5>", "Installation messages in English"],
	//   ["a<enter><wait5>", "Keyboard type: unchanged"],
	//
	//   ["a<enter><wait5>", "Install NetBSD to hard disk"],
	//   ["b<enter><wait5>", "Yes"]
	// ]
	// ```
	//
	// In JSON:
	// ```json
	// {
	//   "boot_steps": [
	//     ["1<enter><wait5>", "Install NetBSD"],
	//     ["a<enter><wait5>", "Installation messages in English"],
	//     ["a<enter><wait5>", "Keyboard type: unchanged"],
	//
	//     ["a<enter><wait5>", "Install NetBSD to hard disk"],
	//     ["b<enter><wait5>", "Yes"]
	//   ]
	// }
	// ```
	BootSteps [][]string `mapstructure:"boot_steps" required:"false"`
	// The size, in megabytes, of the hard disk to create for the VM. By
	// default, this is 40000 (about 40 GB).
	DiskSize uint `mapstructure:"disk_size" required:"false"`
	// Set this to true if you would like to keep the VM registered with
	// UTM. Defaults to false.
	KeepRegistered bool `mapstructure:"keep_registered" required:"false"`
	// Defaults to false. When enabled, Packer will not export the VM. Useful
	// if the build output is not the resultant image, but created inside the
	// VM.
	SkipExport bool `mapstructure:"skip_export" required:"false"`
	// The IP address that should be
	// binded to for VNC. By default packer will use 127.0.0.1 for this. If you
	// wish to bind to all interfaces use 0.0.0.0.
	VNCBindAddress string `mapstructure:"vnc_bind_address" required:"false"`
	// Whether or not to set a password on the VNC server. This option
	// automatically enables the QMP socket. See `qmp_socket_path`. Defaults to
	// `false`.
	VNCUsePassword bool `mapstructure:"vnc_use_password" required:"false"`
	// The minimum and maximum port
	// to use for VNC access to the virtual machine. The builder uses VNC to type
	// the initial boot_command. Because Packer generally runs in parallel,
	// Packer uses a randomly chosen port in this range that appears available. By
	// default this is 5900 to 6000. The minimum and maximum ports are inclusive.
	// The minimum port cannot be set below 5900 due to a quirk in how QEMU parses
	// vnc display address.
	VNCPortMin int `mapstructure:"vnc_port_min" required:"false"`
	VNCPortMax int `mapstructure:"vnc_port_max"`
	// QEMU system architecture of the virtual machine.
	// If this is a QEMU virtual machine, you must specify the architecture
	// Which is required in confirguration. By default, this is aarch64.
	VMArch string `mapstructure:"vm_arch" required:"false"`
	// Backend to use for the virtual machine.
	// apple : Apple Virtualization.framework backend.
	// qemu : QEMU backend.
	// By default, this is qemu.
	VMBackend string `mapstructure:"vm_backend" required:"false"`
	// This is the name of the utm file for the new virtual machine, without
	// the file extension. By default this is packer-BUILDNAME, where
	// "BUILDNAME" is the name of the build.
	VMName string `mapstructure:"vm_name" required:"false"`

	ctx interpolate.Context
}

func (c *Config) Prepare(raws ...interface{}) ([]string, error) {
	err := config.Decode(c, &config.DecodeOpts{
		PluginType:         BuilderId,
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
				"boot_steps",
			},
		},
	}, raws...)
	if err != nil {
		return nil, err
	}

	// Accumulate any errors and warnings
	var errs *packersdk.MultiError
	warnings := make([]string, 0)

	isoWarnings, isoErrs := c.ISOConfig.Prepare(&c.ctx)
	warnings = append(warnings, isoWarnings...)
	errs = packersdk.MultiErrorAppend(errs, isoErrs...)

	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(
		errs, c.OutputConfig.Prepare(&c.ctx, &c.PackerConfig)...)
	errs = packersdk.MultiErrorAppend(errs, c.HTTPConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.CommConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.UtmBundleConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.UtmVersionConfig.Prepare(c.CommConfig.Comm.Type)...)
	errs = packersdk.MultiErrorAppend(errs, c.VNCConfig.Prepare(&c.ctx)...)

	if c.DiskSize == 0 {
		c.DiskSize = 40960
	}

	if c.VMArch == "" {
		c.VMArch = "aarch64"
	}

	if c.VMBackend == "" {
		c.VMBackend = "qemu"
	}
	// Validate and use Enums for the VM backend
	switch c.VMBackend {
	case "apple":
		c.VMBackend = "ApPl"
	case "qemu":
		c.VMBackend = "QeMu"
	default:
		errs = packersdk.MultiErrorAppend(
			errs, errors.New("vm_backend must be either 'apple' or 'qemu'"))
	}

	if c.VNCBindAddress == "" {
		c.VNCBindAddress = "127.0.0.1"
	}

	if c.VNCPortMin == 0 {
		c.VNCPortMin = 5900
	}

	if c.VNCPortMax == 0 {
		c.VNCPortMax = 6000
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf(
			"packer-%s-%d", c.PackerBuildName, interpolate.InitTime.Unix())
	}

	if c.VNCPortMin < 5900 {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf("vnc_port_min cannot be below 5900"))
	}

	if c.VNCPortMin > 65535 || c.VNCPortMax > 65535 {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf("vmc_port_min and vnc_port_max must both be below 65535 to be valid TCP ports"))
	}

	if c.VNCPortMin > c.VNCPortMax {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf("vnc_port_min must be less than vnc_port_max"))
	}

	if len(c.BootCommand) > 0 && len(c.BootSteps) > 0 {
		errs = packersdk.MultiErrorAppend(errs,
			fmt.Errorf("boot_command and boot_steps cannot be used together"))
	}
	// Warnings
	if c.ShutdownCommand == "" {
		warnings = append(warnings,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil

}
