package common

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

var (
	//go:embed scripts/*
	osascripts embed.FS
)

// A driver is able to talk to UTM and perform certain
// operations with it. Some of the operations on here may seem overly
// specific, but they were built specifically in mind to handle features
// of the UTM builder for Packer, and to abstract differences in
// versions out of the builder steps, so sometimes the methods are
// extremely specific.
type Driver interface {
	// Delete a VM by name
	Delete(string) error

	// Executes the given AppleScript with the given arguments.
	ExecuteOsaScript(command ...string) (string, error)

	// Export a VM to a UTM file
	Export(string, string) error

	// Import a VM
	Import(string) (string, error)

	// Checks if the VM with the given id is running.
	IsRunning(string) (bool, error)

	// Get guest tools iso path
	GuestToolsIsoPath() (string, error)

	// Stop stops a running machine, forcefully.
	Stop(string) error

	// Utmctl executes the given Utmctl command
	// and returns the stdout channel as string
	Utmctl(...string) (string, error)
	// UtmctlWithInput sends sensitive input on stdin without placing it in argv.
	UtmctlWithInput(context.Context, string, ...string) (string, error)

	// Verify checks to make sure that this driver should function
	// properly. If there is any indication the driver can't function,
	// this will return an error.
	Verify() error

	// Version reads the version of UTM that is installed.
	Version() (string, error)
}

// NewDriver creates a new driver for UTM.
func NewDriver() (Driver, error) {
	var utmctlPath string

	var err error
	utmctlPath, err = exec.LookPath("utmctl")
	if err != nil {
		return nil, err
	}
	log.Printf("utmctl path: %s", utmctlPath)

	// Get the version of UTM
	var driver Driver
	driver = &Utm45Driver{utmctlPath}
	version, err := driver.Version()
	if err != nil {
		return nil, fmt.Errorf("reading UTM version: %w", err)
	}
	log.Printf("UTM version: %s", version)
	driver, err = driverForVersion(version, utmctlPath)
	if err != nil {
		return nil, err
	}
	if err := driver.Verify(); err != nil {
		return nil, err
	}
	return driver, nil
}

func driverForVersion(version, utmctlPath string) (Driver, error) {
	versionParts := strings.Split(version, ".")
	if len(versionParts) < 2 {
		return nil, fmt.Errorf("invalid UTM version: %q", version)
	}
	major, majorErr := strconv.Atoi(versionParts[0])
	minor, minorErr := strconv.Atoi(versionParts[1])
	if majorErr != nil || minorErr != nil {
		return nil, fmt.Errorf("invalid UTM version: %q", version)
	}
	base := Utm45Driver{utmctlPath}
	switch {
	case major == 4 && minor == 5:
		return &base, nil
	case major == 4 && minor == 6:
		return &Utm46Driver{base}, nil
	case major == 4 && minor == 7:
		return &Utm47Driver{Utm46Driver{base}}, nil
	case major == 5:
		return &Utm50Driver{Utm47Driver{Utm46Driver{base}}}, nil
	default:
		return nil, fmt.Errorf("unsupported UTM version: %s", version)
	}
}
