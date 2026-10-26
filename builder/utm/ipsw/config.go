//go:generate packer-sdc mapstructure-to-hcl2 -type Config
//go:generate packer-sdc struct-markdown

package ipsw

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

const BuilderId = "naveenrajm7.ipsw"

// Config describes a macOS restore build on UTM's Apple backend.
type Config struct {
	common.PackerConfig      `mapstructure:",squash"`
	communicator.Config      `mapstructure:",squash"`
	utmcommon.ExportConfig   `mapstructure:",squash"`
	utmcommon.OutputConfig   `mapstructure:",squash"`
	utmcommon.ShutdownConfig `mapstructure:",squash"`

	// URL or local path to a macOS restore IPSW.
	IPSWURL string `mapstructure:"ipsw_url" required:"true"`
	// Packer checksum for the IPSW, for example sha256:<hex>.
	IPSWChecksum string `mapstructure:"ipsw_checksum" required:"true"`
	// Optional download destination; the Packer cache is used by default.
	TargetPath string `mapstructure:"target_path" required:"false"`
	// Name of the UTM virtual machine.
	VMName string `mapstructure:"vm_name" required:"false"`
	// CPUs assigned to the guest; defaults to 4.
	CPUs int `mapstructure:"cpus" required:"false"`
	// Guest memory in MiB; defaults to 8192.
	Memory int `mapstructure:"memory" required:"false"`
	// Maximum duration of the macOS restore; defaults to 90 minutes.
	InstallTimeout time.Duration `mapstructure:"install_timeout" required:"false"`
	// Keep the VM registered in UTM after a successful export.
	KeepRegistered bool `mapstructure:"keep_registered" required:"false"`
	// Enable automatic login during first-boot provisioning (macOS 27+).
	MacOSAutoLogin bool `mapstructure:"macos_auto_login" required:"false"`

	ctx interpolate.Context
}

func (c *Config) Prepare(raws ...interface{}) ([]string, error) {
	if err := config.Decode(c, &config.DecodeOpts{
		PluginType:         BuilderId,
		Interpolate:        true,
		InterpolateContext: &c.ctx,
	}, raws...); err != nil {
		return nil, err
	}

	var errs *packersdk.MultiError
	if c.Type == "" {
		c.Type = "none"
	}
	if c.Type != "none" && c.Type != "ssh" {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("utm-ipsw supports communicator = 'none' or 'ssh'"))
	}
	if c.Type == "ssh" {
		if c.SSHUsername == "" || c.SSHPassword == "" {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("ssh_username and ssh_password are required for first-boot macOS provisioning"))
		}
	}
	if strings.TrimSpace(c.IPSWURL) == "" || strings.TrimSpace(c.IPSWChecksum) == "" {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("ipsw_url and ipsw_checksum are required"))
	}
	if c.VMName == "" {
		c.VMName = fmt.Sprintf("packer-%s-%d", c.PackerBuildName, interpolate.InitTime.Unix())
	}
	if c.CPUs == 0 {
		c.CPUs = 4
	}
	if c.Memory == 0 {
		c.Memory = 8192
	}
	if c.CPUs < 1 || c.Memory < 1024 {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("cpus must be positive and memory must be at least 1024 MiB"))
	}
	if c.InstallTimeout == 0 {
		c.InstallTimeout = 90 * time.Minute
	}
	if c.InstallTimeout < 0 {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("install_timeout must be positive"))
	}
	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.OutputConfig.Prepare(&c.ctx, &c.PackerConfig)...)
	errs = packersdk.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	if c.Type == "ssh" {
		errs = packersdk.MultiErrorAppend(errs, c.Config.Prepare(&c.ctx)...)
	}
	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}
	return nil, nil
}
