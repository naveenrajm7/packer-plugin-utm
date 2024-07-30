package common

import (
	"log"
	"os/exec"
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

	// Import a VM
	Import(string, string) error

	// Checks if the VM with the given name is running.
	IsRunning(string) (bool, error)

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
	driver := &Utm45Driver{utmctlPath}
	if err := driver.Verify(); err != nil {
		return nil, err
	}

	return driver, nil
}
