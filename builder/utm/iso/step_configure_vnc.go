// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package iso

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/net"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

// This step configures the VM to enable the VNC server.
//
// Uses:
//
//	ui     packersdk.Ui
//
// Produces:
//
//	vnc_port int - The port that VNC is configured to listen on.
type stepConfigureVNC struct {
	Enabled            bool
	VNCBindAddress     string
	VNCPortMin         int
	VNCPortMax         int
	VNCDisablePassword bool

	l *net.Listener
}

func VNCPassword(skipPassword bool) string {
	if skipPassword {
		return ""
	}
	length := int(8)

	charSet := []byte("012345689abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	charSetLength := len(charSet)

	password := make([]byte, length)

	for i := 0; i < length; i++ {
		password[i] = charSet[rand.Intn(charSetLength)]
	}

	return string(password)
}

func (s *stepConfigureVNC) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if !s.Enabled {
		log.Println("[INFO] Skipping VNC configuration step...")
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(utmcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmId := state.Get("vmId").(string)

	// Find an open VNC port. Note that this can still fail later on
	// because we have to release the port at some point. But this does its
	// best.
	msg := fmt.Sprintf("Looking for available port between %d and %d on %s", s.VNCPortMin, s.VNCPortMax, s.VNCBindAddress)
	ui.Say(msg)
	log.Print(msg)

	var err error
	s.l, err = net.ListenRangeConfig{
		Addr:    s.VNCBindAddress,
		Min:     s.VNCPortMin,
		Max:     s.VNCPortMax,
		Network: "tcp",
	}.Listen(ctx)
	if err != nil {
		err := fmt.Errorf("error finding VNC port: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	_ = s.l.Listener.Close() // free port, but don't unlock lock file
	vncPort := s.l.Port

	vncPassword := VNCPassword(s.VNCDisablePassword)

	log.Printf("Found available VNC port: %d on IP: %s", vncPort, s.VNCBindAddress)
	state.Put("vnc_port", vncPort)
	state.Put("vnc_password", vncPassword)

	// Add VNC arguments to the VM via Qemu additional arguments.
	// Send choosen vncPort - 5900 as the VNC port.
	// IMPORTANT: The AppleScript replaces (not appends) all QEMU additional args,
	// so we must include any previously-set user args alongside the VNC arg.
	vncQemuArg := fmt.Sprintf("-vnc %s:%d", s.VNCBindAddress, vncPort-5900)

	// Collect all args: user qemuargs (already set) + VNC arg
	addQemuArgsCommand := []string{
		"add_qemu_additional_args.applescript", vmId,
		"--args",
	}
	// Re-include user qemuargs so they are not overwritten
	if userArgs, ok := state.Get("userQemuArgs").([]string); ok {
		addQemuArgsCommand = append(addQemuArgsCommand, userArgs...)
	}
	addQemuArgsCommand = append(addQemuArgsCommand, vncQemuArg)

	ui.Say("Adding QEMU additional arguments...")
	_, err = driver.ExecuteOsaScript(addQemuArgsCommand...)
	if err != nil {
		err := fmt.Errorf("error adding QEMU additional arguments: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	// Append the VNC QEMU argument to build-time args for later cleanup
	buildTimeArgs, _ := state.Get("buildTimeQemuArgs").([]string)
	buildTimeArgs = append(buildTimeArgs, vncQemuArg)
	state.Put("buildTimeQemuArgs", buildTimeArgs)

	return multistep.ActionContinue
}

func (s *stepConfigureVNC) Cleanup(multistep.StateBag) {
	// release the port
	if s.l != nil {
		err := s.l.Close()
		if err != nil {
			log.Printf("failed to unlock port lockfile: %v", err)
		}
	}
}
