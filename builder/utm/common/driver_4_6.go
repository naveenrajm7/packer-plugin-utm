package common

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// Utm46Driver are inherited from Utm45Driver.
type Utm46Driver struct {
	Utm45Driver
}

// UTM 4.6 : We import a VM by utm file using UTM import command.
func (d *Utm46Driver) Import(path string) (string, error) {
	var stdout bytes.Buffer

	// Import VM
	cmd := exec.Command(
		"osascript", "-e",
		fmt.Sprintf(`tell application %q to import new virtual machine from POSIX file %q`, d.appPath(), path),
	)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Get the output of the command
	output := stdout.String()

	// Regular expression to capture the VM ID
	re := regexp.MustCompile(`virtual machine id ([A-F0-9-]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		vmId := matches[1] // Capture the VM ID
		return vmId, nil
	}

	return "", fmt.Errorf("failed to import VM: %s", output)
}

// Export VM to UTM file
func (d *Utm46Driver) Export(vmId string, path string) error {
	var stdout bytes.Buffer

	// Export VM
	cmd := exec.Command(
		"osascript", "-e",
		fmt.Sprintf(`tell application %q to export virtual machine id %q to POSIX file %q`, d.appPath(), vmId, path),
	)
	// print command to log
	log.Printf("Executing command: %s", cmd.String())
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return err
	}

	log.Printf("Export output: %s", stdout.String())

	return nil
}

// Return the downloaded guest tools path if available.
func (d *Utm46Driver) GuestToolsIsoPath() (string, error) {
	// The default path to the guest tools, where UTM downloads it
	guestToolsPath := filepath.Join(os.Getenv("HOME"), "Library/Containers/com.utmapp.UTM/Data/Library/Application Support/GuestSupportTools/utm-guest-tools-latest.iso")

	// Check if the file exists
	if _, err := os.Stat(guestToolsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("guest tools ISO not found at path: %s", guestToolsPath)
	}

	return guestToolsPath, nil
}
