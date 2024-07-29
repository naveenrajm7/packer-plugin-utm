package common

import (
	"bytes"
	"os/exec"
	"strings"
)

type Utm45Driver struct {
	// This is the path to the utmctl binary
	utmctlPath string
}

func (d *Utm45Driver) Delete(name string) error {
	return d.Utmctl("delete", name)
}

func (d *Utm45Driver) Import(name string, path string) error {
	// TODO: Implement this
	return nil
}

func (d *Utm45Driver) IsRunning(name string) (bool, error) {
	var stdout bytes.Buffer

	cmd := exec.Command(d.utmctlPath, "status", name)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false, err
	}

	output := strings.TrimSpace(stdout.String())
	if output == "started" {
		return true, nil
	}

	// We consider "stopping" to still be running. We wait for it to
	// be completely stopped or some other state.
	if output == "stopping" {
		return true, nil
	}

	// We consider "paused" to still be running. We wait for it to
	// be completely stopped or some other state.
	if output == "paused" {
		return true, nil
	}

	// There might be other intermediate states that we consider
	// running, like "pausing", "resuming", "starting", etc.
	// but for now we just use these three.

	return false, nil
}

func (d *Utm45Driver) Stop(name string) error {
	if err := d.Utmctl("stop", name); err != nil {
		return err
	}
	return nil
}

func (d *Utm45Driver) Utmctl(args ...string) error {
	// TODO: Implement this
	return nil
}

func (d *Utm45Driver) Verify() error {
	return nil
}

func (d *Utm45Driver) Version() (string, error) {
	// TODO: Implement this
	return "4.5", nil
}
