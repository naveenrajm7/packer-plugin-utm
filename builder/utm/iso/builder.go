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

// Builder implements packersdk.Builder and builds the actual UTM
// images starting from an ISO file.
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
		&utmcommon.StepDownloadGuestAdditions{
			GuestAdditionsMode:       b.config.GuestAdditionsMode,
			GuestAdditionsURL:        b.config.GuestAdditionsURL,
			GuestAdditionsSHA256:     b.config.GuestAdditionsSHA256,
			GuestAdditionsTargetPath: b.config.GuestAdditionsTargetPath,
			Ctx:                      b.config.ctx,
		},
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
		&commonsteps.StepCreateFloppy{
			Files:       b.config.FloppyFiles,
			Directories: b.config.FloppyDirectories,
			Label:       b.config.FloppyLabel,
		},
		&commonsteps.StepCreateCD{
			Files:   b.config.CDFiles,
			Content: b.config.CDContent,
			Label:   b.config.CDLabel,
		},
		new(utmcommon.StepHTTPIPDiscover),
		commonsteps.HTTPServerFromHTTPConfig(&b.config.HTTPConfig),
		&utmcommon.StepSshKeyPair{
			Debug:        b.config.PackerDebug,
			DebugKeyPath: fmt.Sprintf("%s.pem", b.config.PackerBuildName),
			Comm:         &b.config.Comm,
		},
		&utmcommon.StepCreateVM{
			VMName:         b.config.VMName,
			VMBackend:      b.config.VMBackend,
			VMArch:         b.config.VMArch,
			VMIcon:         b.config.VMIcon,
			HWConfig:       b.config.HWConfig,
			UEFIBoot:       b.config.UEFIBoot,
			Hypervisor:     b.config.Hypervisor,
			KeepRegistered: b.config.KeepRegistered,
		},
		&utmcommon.StepConfigureQemuArgs{
			QemuArgs: b.config.QemuArgs,
		},
		// TODO: Make sure ISO is first in the list for boot order
		new(stepCreateDisk),
		&utmcommon.StepAttachISOs{
			AttachBootISO:           true, // Attach boot ISO , since CreateVM does not.
			ISOInterface:            b.config.ISOInterface,
			GuestAdditionsMode:      b.config.GuestAdditionsMode,
			GuestAdditionsInterface: b.config.GuestAdditionsInterface,
		},
		new(utmcommon.StepAttachFloppy),
		&utmcommon.StepAttachDisplay{
			HardwareType: b.config.DisplayHardwareType,
		},
		&utmcommon.StepPortForwarding{
			CommConfig:             &b.config.Comm,
			HostPortMin:            b.config.HostPortMin,
			HostPortMax:            b.config.HostPortMax,
			SkipNatMapping:         b.config.SkipNatMapping,
			ClearNetworkInterfaces: true,
		},
		&stepConfigureVNC{
			Enabled:            !b.config.DisableVNC,
			VNCBindAddress:     b.config.VNCBindAddress,
			VNCPortMin:         b.config.VNCPortMin,
			VNCPortMax:         b.config.VNCPortMax,
			VNCDisablePassword: !b.config.VNCUsePassword,
		},
		&utmcommon.StepPause{
			Message: "UTM API Unavailable: Add a display device to the VM for VNC to work",
			NoPause: b.config.DisplayNoPause,
		},
		&utmcommon.StepRun{},
		&stepTypeBootCommand{},
		&utmcommon.StepPause{
			Message: "Confirm Install is complete, VM is running with OS installed. (Next steps is connecting to the VM)",
			NoPause: b.config.BootNoPause,
		},
		// Below three steps are for VMs that require a reboot after install.
		// and also the removal of the ISO file.
		// // We stop the VM to remove the ISO file.
		// &utmcommon.StepStopVm{},
		// // After install is complete, remove the ISO file.
		// // Currently no way to identify the ISO driver, so remove the first disk.
		// &stepRemoveFirstDisk{},
		// // We start the VM again for the next steps.
		// &utmcommon.StepRun{},
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      utmcommon.CommHost(b.config.Comm.Host()),
			SSHConfig: b.config.Comm.SSHConfigFunc(),
			SSHPort:   utmcommon.CommPort,
			WinRMPort: utmcommon.CommPort,
		},
		&utmcommon.StepUploadVersion{
			Path: *b.config.UtmVersionFile,
		},
		// TODO: Add StepUploadGuestAdditions
		new(commonsteps.StepProvision),
		&commonsteps.StepCleanupTempKeys{
			Comm: &b.config.Comm,
		},
		&utmcommon.StepShutdown{
			Command:         b.config.ShutdownCommand,
			Timeout:         b.config.ShutdownTimeout,
			Delay:           b.config.PostShutdownDelay,
			DisableShutdown: b.config.DisableShutdown,
		},
		&utmcommon.StepRemoveDevices{
			Bundling: b.config.UtmBundleConfig,
		},
		&utmcommon.StepPause{
			Message: "Make required changes to the VM before export.\nRemove display, Add Serial port, Icon, etc.",
			NoPause: b.config.ExportNoPause,
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
