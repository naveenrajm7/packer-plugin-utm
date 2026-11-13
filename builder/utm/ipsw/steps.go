package ipsw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

var vmIDPattern = regexp.MustCompile(`[0-9a-fA-F-]{36}`)

type stepCreateVM struct {
	KeepRegistered bool
	vmID           string
}

func (s *stepCreateVM) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	path := state.Get("ipsw_path").(string)
	absPath, err := filepath.Abs(path)
	if err == nil {
		absPath, err = filepath.EvalSymlinks(absPath)
	}
	if err != nil {
		return halt(state, ui, fmt.Errorf("resolving IPSW path: %w", err))
	}
	if _, err := os.Stat(absPath); err != nil {
		return halt(state, ui, fmt.Errorf("reading IPSW: %w", err))
	}
	ui.Say("Creating Apple Virtualization macOS VM...")
	out, err := driver.ExecuteOsaScript("create_ipsw_vm.applescript", config.VMName,
		absPath, fmt.Sprint(config.CPUs), fmt.Sprint(config.Memory))
	if err != nil {
		return halt(state, ui, fmt.Errorf("creating macOS VM: %w", err))
	}
	s.vmID = vmIDPattern.FindString(out)
	if s.vmID == "" {
		return halt(state, ui, fmt.Errorf("UTM did not return a VM ID: %q", out))
	}
	state.Put("vmId", s.vmID)
	state.Put("vmName", config.VMName)
	return multistep.ActionContinue
}

func (s *stepCreateVM) Cleanup(state multistep.StateBag) {
	if s.vmID == "" {
		return
	}
	if _, cancelled := state.GetOk(multistep.StateCancelled); !cancelled {
		if _, halted := state.GetOk(multistep.StateHalted); !halted && s.KeepRegistered {
			return
		}
	}
	if err := state.Get("driver").(utmcommon.Driver).Delete(s.vmID); err != nil {
		state.Get("ui").(packersdk.Ui).Error(fmt.Sprintf("Deleting temporary VM: %v", err))
	}
}

type stepInstall struct {
	Timeout              time.Duration
	PollInterval         time.Duration
	MaxTransientFailures int
}

func (s *stepInstall) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmID := state.Get("vmId").(string)
	ui.Say("Installing macOS from IPSW...")
	if _, err := driver.ExecuteOsaScript("install_ipsw_vm.applescript", vmID); err != nil {
		return halt(state, ui, fmt.Errorf("starting macOS installation: %w", err))
	}
	interval := s.PollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	installCtx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	maxTransientFailures := s.MaxTransientFailures
	if maxTransientFailures <= 0 {
		maxTransientFailures = 12
	}
	transientFailures := 0
	for {
		out, err := driver.ExecuteOsaScript("query_ipsw_install.applescript", vmID)
		if err != nil {
			if !isTransientUTMConnectionError(err) || transientFailures >= maxTransientFailures {
				return halt(state, ui, fmt.Errorf("macOS installation failed: %w", err))
			}
			transientFailures++
			if transientFailures == 1 {
				ui.Message("UTM install status connection interrupted; retrying...")
			}
		} else {
			transientFailures = 0
			if strings.HasPrefix(strings.TrimSpace(out), "false|") {
				ui.Say("macOS installation complete.")
				return multistep.ActionContinue
			}
			if !strings.HasPrefix(strings.TrimSpace(out), "true|") {
				return halt(state, ui, fmt.Errorf("unexpected installation status: %q", out))
			}
		}
		select {
		case <-installCtx.Done():
			return halt(state, ui, fmt.Errorf("waiting for macOS installation: %w", installCtx.Err()))
		case <-time.After(interval):
		}
	}
}

func (*stepInstall) Cleanup(multistep.StateBag) {}

func isTransientUTMConnectionError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "(-600)") || strings.Contains(message, "(-609)")
}

type stepStartProvision struct {
	Username, FullName, Password string
	AutoLogin                    bool
	vmID                         string
}

func (s *stepStartProvision) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmID := state.Get("vmId").(string)
	args := []string{"start", vmID, "--provision-username", s.Username,
		"--provision-full-name", s.FullName, "--provision-password-stdin", "--provision-ssh"}
	if s.AutoLogin {
		args = append(args, "--provision-auto-login")
	}
	ui.Say("Starting macOS with first-boot account and SSH provisioning...")
	s.vmID = vmID
	if _, err := driver.UtmctlWithInput(ctx, s.Password+"\n", args...); err != nil {
		return halt(state, ui, fmt.Errorf("first-boot macOS provisioning: %w", err))
	}
	state.Put("instance_id", vmID)
	return multistep.ActionContinue
}

func (s *stepStartProvision) Cleanup(state multistep.StateBag) {
	if s.vmID == "" {
		return
	}
	driver := state.Get("driver").(utmcommon.Driver)
	if running, err := driver.IsRunning(s.vmID); err == nil && running {
		if err := driver.Stop(s.vmID); err != nil {
			state.Get("ui").(packersdk.Ui).Error(fmt.Sprintf("Stopping VM: %v", err))
		}
	}
}

type stepRequestShutdown struct {
	Timeout       time.Duration
	Delay         time.Duration
	PollInterval  time.Duration
	FallbackAfter time.Duration
	Password      string
}

func (s *stepRequestShutdown) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmID := state.Get("vmId").(string)
	ui.Say("Requesting macOS guest shutdown...")
	if _, err := driver.Utmctl("stop", vmID, "--request"); err != nil {
		return halt(state, ui, fmt.Errorf("requesting guest shutdown: %w", err))
	}
	interval := s.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	fallbackAfter := s.FallbackAfter
	if fallbackAfter <= 0 {
		fallbackAfter = 30 * time.Second
	}
	fallbackAt := time.Now().Add(fallbackAfter)
	fallbackAttempted := false
	var fallbackErr error
	shutdownCtx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	for {
		running, err := driver.IsRunning(vmID)
		if err != nil {
			return halt(state, ui, fmt.Errorf("checking guest shutdown: %w", err))
		}
		if !running {
			if s.Delay > 0 {
				select {
				case <-shutdownCtx.Done():
					return halt(state, ui, fmt.Errorf("waiting for guest shutdown: %w", shutdownCtx.Err()))
				case <-time.After(s.Delay):
				}
			}
			return multistep.ActionContinue
		}
		if !fallbackAttempted && time.Now().After(fallbackAt) && s.Password != "" {
			fallbackAttempted = true
			ui.Message("UTM shutdown request was ignored; asking macOS to shut down over SSH...")
			comm, ok := state.Get("communicator").(packersdk.Communicator)
			if !ok {
				fallbackErr = fmt.Errorf("SSH communicator is unavailable")
			} else {
				cmd := &packersdk.RemoteCmd{
					Command: "sudo -S -p '' /sbin/shutdown -h now",
					Stdin:   strings.NewReader(s.Password + "\n"),
				}
				fallbackErr = comm.Start(shutdownCtx, cmd)
			}
		}
		select {
		case <-shutdownCtx.Done():
			if fallbackErr != nil {
				return halt(state, ui, fmt.Errorf("waiting for guest shutdown: %w (SSH fallback: %v)", shutdownCtx.Err(), fallbackErr))
			}
			return halt(state, ui, fmt.Errorf("waiting for guest shutdown: %w", shutdownCtx.Err()))
		case <-time.After(interval):
		}
	}
}

func (*stepRequestShutdown) Cleanup(multistep.StateBag) {}

func halt(state multistep.StateBag, ui packersdk.Ui, err error) multistep.StepAction {
	state.Put("error", err)
	ui.Error(err.Error())
	return multistep.ActionHalt
}
