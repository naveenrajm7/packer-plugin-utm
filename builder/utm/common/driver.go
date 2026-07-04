package common

import (
	"embed"
	"fmt"
	"log"
	"os/exec"
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
		log.Fatalf("Error getting UTM version: %v", err)
	}
	fmt.Printf("UTM version: %s\n", version)

	// Parse the version to get major and minor parts
	versionParts := strings.Split(version, ".")
	if len(versionParts) < 2 {
		log.Fatalf("Invalid UTM version format: %s", version)
	}
	majorMinorVersion := fmt.Sprintf("%s.%s", versionParts[0], versionParts[1])

	// Decide which driver to use based on the version
	switch majorMinorVersion {
	case "4.5":
		driver = &Utm45Driver{utmctlPath}
	case "4.6":
		driver = &Utm46Driver{Utm45Driver{utmctlPath}}
	case "4.7":
		driver = &Utm47Driver{Utm46Driver{Utm45Driver{utmctlPath}}}
	case "5.0":
		driver = &Utm50Driver{Utm47Driver{Utm46Driver{Utm45Driver{utmctlPath}}}}
	default:
		log.Fatalf("Unsupported UTM version: %s", version)
	}

	if err := driver.Verify(); err != nil {
		return nil, err
	}

	return driver, nil
}
