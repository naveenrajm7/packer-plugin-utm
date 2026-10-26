package ipsw

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

const testVMID = "12345678-1234-1234-1234-123456789abc"

func ipswState() (multistep.StateBag, *utmcommon.DriverMock) {
	state := new(multistep.BasicStateBag)
	driver := new(utmcommon.DriverMock)
	state.Put("driver", driver)
	state.Put("ui", &packersdk.BasicUi{Reader: new(bytes.Buffer), Writer: new(bytes.Buffer)})
	return state, driver
}

func TestCreateUsesIPSW(t *testing.T) {
	path := t.TempDir() + "/restore.ipsw"
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	state, driver := ipswState()
	state.Put("config", &Config{VMName: "mac-test", CPUs: 4, Memory: 8192})
	state.Put("ipsw_path", path)
	driver.ExecuteOsaResult = testVMID
	step := new(stepCreateVM)
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("create halted: %v", state.Get("error"))
	}
	call := driver.ExecuteOsaCalls[0]
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if call[0] != "create_ipsw_vm.applescript" || call[2] != resolved || call[3] != "4" || call[4] != "8192" {
		t.Fatalf("unexpected create command: %#v", call)
	}
	if state.Get("vmId") != testVMID {
		t.Fatal("VM ID not stored")
	}
	step.Cleanup(state)
	if !driver.DeleteCalled {
		t.Fatal("temporary VM was not deleted")
	}
}

func TestProvisionPasswordOnlyOnStdin(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	step := &stepStartProvision{Username: "packer", FullName: "Packer", Password: "secret-value"}
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("provision halted: %v", state.Get("error"))
	}
	args := strings.Join(driver.UtmctlInputCalls[0], " ")
	if strings.Contains(args, "secret-value") || !strings.Contains(args, "--provision-password-stdin") {
		t.Fatalf("unsafe provisioning command: %s", args)
	}
	if driver.UtmctlInput != "secret-value\n" {
		t.Fatal("password was not sent on stdin")
	}
}

func TestInstallPollsUntilComplete(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	driver.ExecuteOsaResults = []string{"", "true|0.4", "false|1.0"}
	step := &stepInstall{Timeout: time.Second, PollInterval: time.Millisecond}
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("install halted: %v", state.Get("error"))
	}
	if len(driver.ExecuteOsaCalls) != 3 || driver.ExecuteOsaCalls[2][0] != "query_ipsw_install.applescript" {
		t.Fatalf("unexpected install calls: %#v", driver.ExecuteOsaCalls)
	}
}

func TestInstallRetriesTransientUTMConnectionFailure(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	driver.ExecuteOsaErrs = []error{nil, errors.New("Connection is invalid. (-609)"), nil}
	driver.ExecuteOsaResults = []string{"", "", "false|1.0"}
	step := &stepInstall{Timeout: time.Second, PollInterval: time.Millisecond}
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("transient connection failure halted install: %v", state.Get("error"))
	}
}

func TestInstallFailsPersistentUTMConnectionFailure(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	driver.ExecuteOsaErrs = []error{nil, errors.New("Connection is invalid. (-609)"), errors.New("Connection is invalid. (-609)")}
	step := &stepInstall{Timeout: time.Second, PollInterval: time.Millisecond, MaxTransientFailures: 1}
	if action := step.Run(context.Background(), state); action != multistep.ActionHalt {
		t.Fatal("persistent connection failure did not halt install")
	}
}

func TestInstallRetriesUTMAppNotRunningError(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	driver.ExecuteOsaErrs = []error{nil, errors.New("Application isn't running. (-600)"), nil}
	driver.ExecuteOsaResults = []string{"", "", "false|1.0"}
	step := &stepInstall{Timeout: time.Second, PollInterval: time.Millisecond}
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("transient app connection failure halted install: %v", state.Get("error"))
	}
}

func TestMacOSShutdownRequestsGuestPowerOff(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	step := &stepRequestShutdown{Timeout: time.Second, PollInterval: time.Millisecond}
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("shutdown halted: %v", state.Get("error"))
	}
	call := driver.UtmctlCalls[0]
	if len(call) != 3 || call[0] != "stop" || call[1] != testVMID || call[2] != "--request" {
		t.Fatalf("unexpected shutdown command: %#v", call)
	}
}

func TestMacOSShutdownFallsBackToGuestSSH(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	driver.IsRunningReturn = true
	comm := new(packersdk.MockCommunicator)
	state.Put("communicator", comm)
	go func() {
		time.Sleep(10 * time.Millisecond)
		driver.Lock()
		driver.IsRunningReturn = false
		driver.Unlock()
	}()
	step := &stepRequestShutdown{
		Timeout: time.Second, PollInterval: time.Millisecond,
		FallbackAfter: time.Millisecond, Password: "test-secret",
	}
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("shutdown halted: %v", state.Get("error"))
	}
	if !comm.StartCalled || comm.StartCmd.Command != "sudo -S -p '' /sbin/shutdown -h now" {
		t.Fatalf("SSH fallback was not started: %#v", comm.StartCmd)
	}
	if strings.Contains(comm.StartCmd.Command, "test-secret") {
		t.Fatal("password appeared in shutdown command")
	}
}
