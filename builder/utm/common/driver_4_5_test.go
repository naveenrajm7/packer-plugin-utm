// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUtm45Driver_impl(t *testing.T) {
	var _ Driver = new(Utm45Driver)
}

func TestUtm45DriverVersionUsesCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "utmctl")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '5.0.8\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	driver := &Utm45Driver{UtmctlPath: path}
	version, err := driver.Version()
	if err != nil || version != "5.0.8" {
		t.Fatalf("Version() = %q, %v; want 5.0.8", version, err)
	}
}

func TestUtm45DriverAppPathFollowsCLI(t *testing.T) {
	app := filepath.Join(t.TempDir(), "UTM.app")
	binDir := filepath.Join(app, "Contents", "MacOS")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(binDir, "utmctl")
	if err := os.WriteFile(bin, []byte(""), 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "utmctl")
	if err := os.Symlink(bin, link); err != nil {
		t.Fatal(err)
	}
	driver := &Utm45Driver{UtmctlPath: link}
	canonicalApp, err := filepath.EvalSymlinks(app)
	if err != nil {
		t.Fatal(err)
	}
	if got := driver.appPath(); got != canonicalApp {
		t.Fatalf("appPath() = %q, want %q", got, canonicalApp)
	}
}

func TestExecuteOsaScriptIncludesStderr(t *testing.T) {
	binDir := t.TempDir()
	fakeOSA := filepath.Join(binDir, "osascript")
	if err := os.WriteFile(fakeOSA, []byte("#!/bin/sh\nprintf 'restore failed' >&2\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	driver := &Utm45Driver{UtmctlPath: "utmctl"}
	_, err := driver.ExecuteOsaScript("install_ipsw_vm.applescript", "test")
	if err == nil || !strings.Contains(err.Error(), "restore failed") {
		t.Fatalf("expected osascript stderr, got %v", err)
	}
}
