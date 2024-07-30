package utm

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

// Builder implements packersdk.Builder and builds the actual UTM
// images.
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

// Run executes a Packer build and returns a packersdk.Artifact representing
// a UTM appliance.
func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	// Create the driver that we'll use to communicate with UTM
	driver, err := utmcommon.NewDriver()
	if err != nil {
		return nil, fmt.Errorf("Failed creating UTM driver: %s", err)
	}

	// Set up the state
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)

	// Build the steps
	steps := []multistep.Step{
		&commonsteps.StepOutputDir{
			Force: b.config.PackerForce,
			Path:  b.config.OutputDir,
		},
		&utmcommon.StepSshKeyPair{
			Debug:        b.config.PackerDebug,
			DebugKeyPath: fmt.Sprintf("%s.pem", b.config.PackerBuildName),
			Comm:         &b.config.Comm,
		},
		&utmcommon.StepDownload{
			Checksum:    b.config.Checksum,
			Description: "UTM",
			Extension:   "utm",
			ResultKey:   "vm_path",
			TargetPath:  b.config.TargetPath,
			Url:         []string{b.config.SourcePath},
		},
		&StepImport{
			Name:           b.config.VMName,
			KeepRegistered: b.config.KeepRegistered,
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

	// Run the steps.
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

	// Report any errors.
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
	return utmcommon.NewArtifact(b.config.OutputDir, generatedData)
}
