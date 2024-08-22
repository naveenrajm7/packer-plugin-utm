package iso

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

const BuilderId = "naveenrajm7.iso"

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	warnings, errs := b.config.Prepare(raws...)
	if errs != nil {
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

	// Setup the state bag
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)

	// Build the steps.
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

	// Run the steps
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
