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
	// driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	// vmName := state.Get("vmName").(string)

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
			err := fmt.Errorf("Error creating port forwarding rule: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		s.l.Listener.Close() // free port, but don't unlock lock file
		commHostPort = s.l.Port

		// Make sure to configure the network interface to 'Emulated VLAN' mode
		// TODO: Add code to configure the network interface 1 to 'Emulated VLAN' mode

		// Add access to localhost => UTM 'Shared Network' interface if necessary
		// TODO: Add code to add access to localhost => UTM 'Shared Network' interface if necessary

		// Create a forwarded port mapping to the VM (on the 'Emulated VLAN' interface)
		// TODO: Add code to create a forwarded port mapping to the VM (on the 'Emulated VLAN' interface)
		// "tcp, 127.0.0.1, hostPort, guestPort"

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
