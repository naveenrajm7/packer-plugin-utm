package common

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/net"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step adds a Emulated VLAN port forwarding definition so that SSH (or WinRM ?)
// is available on the guest machine.
//
// Uses:
//
//	driver Driver
//	ui packersdk.Ui
//	vmName string
//
// Produces:
type StepPortForwarding struct {
	CommConfig     *communicator.Config
	HostPortMin    int
	HostPortMax    int
	SkipNatMapping bool

	l *net.Listener
}

func (s *StepPortForwarding) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmName := state.Get("vmName").(string)

	if s.CommConfig.Type == "none" {
		log.Printf("Not using a communicator, skipping setting up port forwarding...")
		state.Put("commHostPort", 0)
		return multistep.ActionContinue
	}

	guestPort := s.CommConfig.Port()
	commHostPort := guestPort
	if !s.SkipNatMapping {
		log.Printf("Looking for available communicator (SSH, WinRM, etc) port between %d and %d",
			s.HostPortMin, s.HostPortMax)

		var err error
		s.l, err = net.ListenRangeConfig{
			Addr:    "127.0.0.1",
			Min:     s.HostPortMin,
			Max:     s.HostPortMax,
			Network: "tcp",
		}.Listen(ctx)
		if err != nil {
			err := fmt.Errorf("error creating port forwarding rule: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		s.l.Listener.Close() // free port, but don't unlock lock file
		commHostPort = s.l.Port

		// Make sure to clear the network interfaces and prepare for the new configuration
		if _, err := driver.ExecuteOsaScript("clear_network_interfaces.applescript", vmName); err != nil {
			err := fmt.Errorf("error clearing network interfaces: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// We now hard code interfaces as needed by Vagrant,
		// 0 index - 'Shared Network' interface
		// 1 index - 'Emulated VLAN' interface
		// but this should be configurable

		// Add access to localhost => UTM 'Shared Network' interface
		if _, err := driver.ExecuteOsaScript("add_network_interface.applescript", vmName, "ShRd"); err != nil {
			err := fmt.Errorf("error adding network interface: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Make sure to configure the network interface to 'Emulated VLAN' mode
		// required for port forwarding now in packer , later in vagrant
		if _, err := driver.ExecuteOsaScript("add_network_interface.applescript", vmName, "EmUd"); err != nil {
			err := fmt.Errorf("error adding network interface: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Create a forwarded port mapping to the VM (on the 'Emulated VLAN' interface)
		// "tcp, 127.0.0.1, hostPort, guestPort"
		ui.Say(fmt.Sprintf("Creating forwarded port mapping for communicator (SSH, WinRM, etc) (host port %d)", commHostPort))
		command := []string{
			"add_port_forwards.applescript", vmName,
			"--index", "1",
			fmt.Sprintf("TcPp,,%d,127.0.0.1,%d", guestPort, commHostPort),
		}
		if _, err := driver.ExecuteOsaScript(command...); err != nil {
			err := fmt.Errorf("error adding port forwarding rule: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

	}
	// Save the port we're using so that future steps can use it
	state.Put("commHostPort", commHostPort)

	return multistep.ActionContinue
}

func (s *StepPortForwarding) Cleanup(state multistep.StateBag) {
	if s.l != nil {
		err := s.l.Close()
		if err != nil {
			log.Printf("failed to unlock port lockfile: %v", err)
		}
	}
}
