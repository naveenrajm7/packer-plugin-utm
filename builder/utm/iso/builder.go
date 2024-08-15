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
	bootcommand.BootConfig     `mapstructure:",squash"`
	utmcommon.ExportConfig     `mapstructure:",squash"`
	utmcommon.OutputConfig     `mapstructure:",squash"`
	utmcommon.ShutdownConfig   `mapstructure:",squash"`
	utmcommon.CommConfig       `mapstructure:",squash"`
	utmcommon.UtmVersionConfig `mapstructure:",squash"`
	utmcommon.UtmBundleConfig  `mapstructure:",squash"`

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
	errs = packersdk.MultiErrorAppend(errs, b.config.BootConfig.Prepare(&b.config.ctx)...)

	if b.config.DiskSize == 0 {
		b.config.DiskSize = 40000
	}

	if b.config.VMName == "" {
		b.config.VMName = fmt.Sprintf(
			"packer-%s-%d", b.config.PackerBuildName, interpolate.InitTime.Unix())
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
		return nil, fmt.Errorf("Failed creating UTM driver: %s", err)
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
		&utmcommon.StepPortForwarding{
			CommConfig:     &b.config.CommConfig.Comm,
			HostPortMin:    b.config.HostPortMin,
			HostPortMax:    b.config.HostPortMax,
			SkipNatMapping: b.config.SkipNatMapping,
		},
		&utmcommon.StepRun{},
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
		return nil, errors.New("Build was cancelled.")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	generatedData := map[string]interface{}{"generated_data": state.Get("generated_data")}
	return utmcommon.NewArtifact(b.config.OutputDir, b.config.VMName, generatedData)
}
