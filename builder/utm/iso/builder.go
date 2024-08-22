//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package iso

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

const BuilderId = "naveenrajm7.iso"

type Builder struct {
	config Config
	runner multistep.Runner
}

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

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	err := config.Decode(&b.config, &config.DecodeOpts{
		PluginType:         utmcommon.BuilderId, // "naveenrajm7.utm"
		Interpolate:        true,
		InterpolateContext: &b.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
			},
		},
	}, raws...)
	if err != nil {
		return nil, nil, err
	}

	// Accumulate any errors and warnings
	var errs *packersdk.MultiError
	warnings := make([]string, 0)

	isoWarnings, isoErrs := b.config.ISOConfig.Prepare(&b.config.ctx)
	warnings = append(warnings, isoWarnings...)
	errs = packersdk.MultiErrorAppend(errs, isoErrs...)

	errs = packersdk.MultiErrorAppend(errs, b.config.ExportConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.ExportConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(
		errs, b.config.OutputConfig.Prepare(&b.config.ctx, &b.config.PackerConfig)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.HTTPConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.ShutdownConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.CommConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.UtmBundleConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.UtmVersionConfig.Prepare(b.config.CommConfig.Comm.Type)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.VNCConfig.Prepare(&b.config.ctx)...)

	if b.config.DiskSize == 0 {
		b.config.DiskSize = 40000
	}

	if b.config.VMArch == "" {
		b.config.VMArch = "aarch64"
	}

	if b.config.VMBackend == "" {
		b.config.VMBackend = "qemu"
	}
	// Validate and use Enums for the VM backend
	switch b.config.VMBackend {
	case "apple":
		b.config.VMBackend = "ApPl"
	case "qemu":
		b.config.VMBackend = "QeMu"
	default:
		errs = packersdk.MultiErrorAppend(
			errs, errors.New("vm_backend must be either 'apple' or 'qemu'"))
	}

	if b.config.VNCBindAddress == "" {
		b.config.VNCBindAddress = "127.0.0.1"
	}

	if b.config.VNCPortMin == 0 {
		b.config.VNCPortMin = 5900
	}

	if b.config.VNCPortMax == 0 {
		b.config.VNCPortMax = 6000
	}

	if b.config.VMName == "" {
		b.config.VMName = fmt.Sprintf(
			"packer-%s-%d", b.config.PackerBuildName, interpolate.InitTime.Unix())
	}

	if b.config.VNCPortMin < 5900 {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf("vnc_port_min cannot be below 5900"))
	}

	if b.config.VNCPortMin > 65535 || b.config.VNCPortMax > 65535 {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf("vmc_port_min and vnc_port_max must both be below 65535 to be valid TCP ports"))
	}

	if b.config.VNCPortMin > b.config.VNCPortMax {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf("vnc_port_min must be less than vnc_port_max"))
	}

	if len(b.config.BootCommand) > 0 && len(b.config.BootSteps) > 0 {
		errs = packersdk.MultiErrorAppend(errs,
			fmt.Errorf("boot_command and boot_steps cannot be used together"))
	}
	// Warnings
	if b.config.ShutdownCommand == "" {
		warnings = append(warnings,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, warnings, errs
	}

	return nil, warnings, nil
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	// Create the driver that we'll use to communicate with UTM
	driver, err := utmcommon.NewDriver()
	if err != nil {
		return nil, fmt.Errorf("failed creating UTM driver: %s", err)
	}

	steps := []multistep.Step{
		&commonsteps.StepDownload{
			Checksum:    b.config.ISOChecksum,
			Description: "ISO",
			Extension:   b.config.TargetExtension,
			ResultKey:   "iso_path",
			TargetPath:  b.config.TargetPath,
			Url:         b.config.ISOUrls,
		},
		&commonsteps.StepOutputDir{
			Force: b.config.PackerForce,
			Path:  b.config.OutputDir,
		},
		new(utmcommon.StepHTTPIPDiscover),
		commonsteps.HTTPServerFromHTTPConfig(&b.config.HTTPConfig),
		&utmcommon.StepSshKeyPair{
			Debug:        b.config.PackerDebug,
			DebugKeyPath: fmt.Sprintf("%s.pem", b.config.PackerBuildName),
			Comm:         &b.config.Comm,
		},
		new(stepCreateVM),
		new(stepCreateDisk),
		&utmcommon.StepAttachISOs{
			AttachBootISO: true,
		},
		&utmcommon.StepPause{
			Message: "UTM Bug: Update ISO with same ISO, so we don't get file not found error",
		},
		&utmcommon.StepPortForwarding{
			CommConfig:             &b.config.CommConfig.Comm,
			HostPortMin:            b.config.HostPortMin,
			HostPortMax:            b.config.HostPortMax,
			SkipNatMapping:         b.config.SkipNatMapping,
			ClearNetworkInterfaces: true,
		},
		new(stepConfigureVNC),
		&utmcommon.StepPause{
			Message: "UTM API: Add QEMU Additional Arguments `-vnc 127.0.0.1:port` port=VncPort-5900",
		},
		&utmcommon.StepRun{},
		&stepTypeBootCommand{},
		&communicator.StepConnect{
			Config:    &b.config.CommConfig.Comm,
			Host:      utmcommon.CommHost(b.config.CommConfig.Comm.Host()),
			SSHConfig: b.config.CommConfig.Comm.SSHConfigFunc(),
			SSHPort:   utmcommon.CommPort,
			WinRMPort: utmcommon.CommPort,
		},
		&utmcommon.StepUploadVersion{
			Path: *b.config.UtmVersionFile,
		},
		new(commonsteps.StepProvision),
		&commonsteps.StepCleanupTempKeys{
			Comm: &b.config.CommConfig.Comm,
		},
		&utmcommon.StepShutdown{
			Command:         b.config.ShutdownCommand,
			Timeout:         b.config.ShutdownTimeout,
			Delay:           b.config.PostShutdownDelay,
			DisableShutdown: b.config.DisableShutdown,
		},
		&utmcommon.StepExport{
			Format:         b.config.Format,
			OutputDir:      b.config.OutputDir,
			OutputFilename: b.config.OutputFilename,
			SkipNatMapping: b.config.SkipNatMapping,
			SkipExport:     b.config.SkipExport,
		},
	}

	// Setup the state bag
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)

	// Run
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("build was cancelled")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("build was halted")
	}

	generatedData := map[string]interface{}{"generated_data": state.Get("generated_data")}
	return utmcommon.NewArtifact(b.config.OutputDir, b.config.VMName, generatedData)
}
