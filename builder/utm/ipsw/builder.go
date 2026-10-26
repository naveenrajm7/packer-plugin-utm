package ipsw

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	warnings, err := b.config.Prepare(raws...)
	return nil, warnings, err
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("utm-ipsw requires macOS")
	}
	if b.config.Type == "ssh" {
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err != nil {
			return nil, fmt.Errorf("checking macOS host version: %w", err)
		}
		major, err := strconv.Atoi(strings.Split(strings.TrimSpace(string(out)), ".")[0])
		if err != nil || major < 27 {
			return nil, errors.New("first-boot SSH provisioning requires macOS 27 or later on host and guest; use communicator = 'none' for an artifact-only build")
		}
	}
	driver, err := utmcommon.NewDriver()
	if err != nil {
		return nil, fmt.Errorf("creating UTM driver: %w", err)
	}
	version, err := driver.Version()
	if err != nil {
		return nil, fmt.Errorf("checking UTM version: %w", err)
	}
	if !supportsIPSW(version) {
		return nil, fmt.Errorf("utm-ipsw requires UTM 5.0.6 or later (found %s)", version)
	}

	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)
	state.Put("communicator_config", &b.config.Config)
	state.Put("commHostPort", 0)

	steps := []multistep.Step{
		&commonsteps.StepDownload{
			Checksum: b.config.IPSWChecksum, Description: "macOS restore IPSW",
			Extension: "ipsw", ResultKey: "ipsw_path", TargetPath: b.config.TargetPath,
			Url: []string{b.config.IPSWURL},
		},
		&commonsteps.StepOutputDir{Force: b.config.PackerForce, Path: b.config.OutputDir},
		&stepCreateVM{KeepRegistered: b.config.KeepRegistered},
		&stepInstall{Timeout: b.config.InstallTimeout},
	}
	if b.config.Type == "ssh" {
		steps = append(steps,
			&stepStartProvision{Username: b.config.SSHUsername, FullName: b.config.SSHUsername,
				Password: b.config.SSHPassword, AutoLogin: b.config.MacOSAutoLogin},
			&communicator.StepConnect{
				Config: &b.config.Config, Host: macOSSSHHost(b.config.SSHHost),
				SSHConfig: b.config.Config.SSHConfigFunc(),
				SSHPort:   func(multistep.StateBag) (int, error) { return b.config.Config.Port(), nil },
			},
		)
	}
	// Always run the Packer provision hook. With communicator = "none", guest
	// provisioners must fail explicitly instead of being silently skipped.
	steps = append(steps, new(commonsteps.StepProvision))
	if b.config.Type == "ssh" {
		if b.config.ShutdownCommand != "" || b.config.DisableShutdown {
			steps = append(steps, &utmcommon.StepShutdown{
				Command: b.config.ShutdownCommand, Timeout: b.config.ShutdownTimeout,
				Delay: b.config.PostShutdownDelay, DisableShutdown: b.config.DisableShutdown,
			})
		} else {
			steps = append(steps, &stepRequestShutdown{
				Timeout: b.config.ShutdownTimeout, Delay: b.config.PostShutdownDelay,
				PollInterval: time.Second, Password: b.config.SSHPassword,
			})
		}
	}
	steps = append(steps, &utmcommon.StepExport{
		Format: b.config.Format, OutputDir: b.config.OutputDir,
		OutputFilename: b.config.OutputFilename, SkipNatMapping: true,
	})

	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("build was cancelled")
	}
	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("build was halted")
	}
	return utmcommon.NewArtifact(b.config.OutputDir, b.config.VMName,
		map[string]interface{}{"generated_data": state.Get("generated_data")})
}

func supportsIPSW(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	major, e1 := strconv.Atoi(parts[0])
	minor, e2 := strconv.Atoi(parts[1])
	patch, e3 := strconv.Atoi(parts[2])
	return e1 == nil && e2 == nil && e3 == nil && (major > 5 || (major == 5 && (minor > 0 || patch >= 6)))
}
