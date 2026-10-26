package ipsw

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

func TestMacOSSSHHostDiscoversIPv4(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	driver.UtmctlResult = "fe80::1\n192.168.64.2\n"
	host, err := macOSSSHHost("")(state)
	if err != nil || host != "192.168.64.2" {
		t.Fatalf("host = %q, %v", host, err)
	}
	if len(driver.UtmctlCalls) != 1 || driver.UtmctlCalls[0][0] != "ip-address" {
		t.Fatalf("unexpected utmctl calls: %#v", driver.UtmctlCalls)
	}
}

func TestMacOSSSHHostNotReadyAndOverride(t *testing.T) {
	state, driver := ipswState()
	state.Put("vmId", testVMID)
	if _, err := macOSSSHHost("")(state); err == nil {
		t.Fatal("expected an error while guest IP is unavailable")
	}
	host, err := macOSSSHHost("example.test")(state)
	if err != nil || host != "example.test" {
		t.Fatalf("override = %q, %v", host, err)
	}
	if len(driver.UtmctlCalls) != 1 {
		t.Fatalf("override queried UTM: %#v", driver.UtmctlCalls)
	}
}

func TestMacOSSSHHostIntegration(t *testing.T) {
	vmID := os.Getenv("UTM_TEST_VM_ID")
	if vmID == "" {
		t.Skip("set UTM_TEST_VM_ID to query a running UTM VM")
	}
	driver, err := utmcommon.NewDriver()
	if err != nil {
		t.Fatal(err)
	}
	state := new(multistep.BasicStateBag)
	state.Put("driver", driver)
	state.Put("vmId", vmID)
	host, err := macOSSSHHost("")(state)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("UTM guest IP: %s", host)
}

func TestMacOSSSHConnectIntegration(t *testing.T) {
	vmID := os.Getenv("UTM_TEST_VM_ID")
	password := os.Getenv("UTM_TEST_SSH_PASSWORD")
	if vmID == "" || password == "" {
		t.Skip("set UTM_TEST_VM_ID and UTM_TEST_SSH_PASSWORD")
	}
	driver, err := utmcommon.NewDriver()
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if _, err := cfg.Prepare(map[string]interface{}{
		"ipsw_url": "dummy", "ipsw_checksum": "none",
		"communicator": "ssh", "ssh_username": "packer", "ssh_password": password,
		"ssh_timeout": "30s",
	}); err != nil {
		t.Fatal(err)
	}
	if host := os.Getenv("UTM_TEST_SSH_HOST"); host != "" {
		cfg.SSHHost = host
	}
	if port := os.Getenv("UTM_TEST_SSH_PORT"); port != "" {
		parsed, err := strconv.Atoi(port)
		if err != nil {
			t.Fatal(err)
		}
		cfg.SSHPort = parsed
	}
	state := new(multistep.BasicStateBag)
	state.Put("driver", driver)
	state.Put("vmId", vmID)
	state.Put("ui", &packersdk.BasicUi{Reader: new(bytes.Buffer), Writer: os.Stdout})
	step := &communicator.StepConnect{
		Config: &cfg.Config, Host: macOSSSHHost(cfg.SSHHost),
		SSHConfig: cfg.Config.SSHConfigFunc(),
		SSHPort:   func(multistep.StateBag) (int, error) { return cfg.Config.Port(), nil },
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if action := step.Run(ctx, state); action != multistep.ActionContinue {
		t.Fatalf("connect halted: %v", state.Get("error"))
	}
	t.Log("Packer SSH communicator connected")
}

func TestMacOSShutdownIntegration(t *testing.T) {
	if os.Getenv("UTM_TEST_SHUTDOWN") != "1" {
		t.Skip("set UTM_TEST_SHUTDOWN=1 to stop a running test VM")
	}
	vmID := os.Getenv("UTM_TEST_VM_ID")
	password := os.Getenv("UTM_TEST_SSH_PASSWORD")
	host := os.Getenv("UTM_TEST_SSH_HOST")
	portText := os.Getenv("UTM_TEST_SSH_PORT")
	if vmID == "" || password == "" || host == "" || portText == "" {
		t.Fatal("UTM_TEST_VM_ID, UTM_TEST_SSH_PASSWORD, UTM_TEST_SSH_HOST, and UTM_TEST_SSH_PORT are required")
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	driver, err := utmcommon.NewDriver()
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if _, err := cfg.Prepare(map[string]interface{}{
		"ipsw_url": "dummy", "ipsw_checksum": "none",
		"communicator": "ssh", "ssh_username": "packer", "ssh_password": password,
		"ssh_host": host, "ssh_port": port, "ssh_timeout": "1m",
	}); err != nil {
		t.Fatal(err)
	}
	state := new(multistep.BasicStateBag)
	state.Put("driver", driver)
	state.Put("vmId", vmID)
	state.Put("ui", &packersdk.BasicUi{Reader: new(bytes.Buffer), Writer: os.Stdout})
	connect := &communicator.StepConnect{
		Config: &cfg.Config, Host: macOSSSHHost(cfg.SSHHost),
		SSHConfig: cfg.Config.SSHConfigFunc(),
		SSHPort:   func(multistep.StateBag) (int, error) { return cfg.Config.Port(), nil },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if action := connect.Run(ctx, state); action != multistep.ActionContinue {
		t.Fatalf("connect halted: %v", state.Get("error"))
	}
	shutdown := &stepRequestShutdown{
		Timeout: time.Minute, PollInterval: time.Second,
		FallbackAfter: 3 * time.Second, Password: password,
	}
	if action := shutdown.Run(ctx, state); action != multistep.ActionContinue {
		t.Fatalf("shutdown halted: %v", state.Get("error"))
	}
	t.Log("macOS test VM shut down")
}
