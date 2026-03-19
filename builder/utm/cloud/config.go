//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package cloud

import (
	"errors"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

// Config is the configuration structure for the UTM Cloud builder.
type Config struct {
	common.PackerConfig            `mapstructure:",squash"`
	commonsteps.HTTPConfig         `mapstructure:",squash"`
	commonsteps.ISOConfig          `mapstructure:",squash"`
	commonsteps.CDConfig           `mapstructure:",squash"`
	utmcommon.ExportConfig         `mapstructure:",squash"`
	utmcommon.OutputConfig         `mapstructure:",squash"`
	utmcommon.ShutdownConfig       `mapstructure:",squash"`
	utmcommon.CommConfig           `mapstructure:",squash"`
	utmcommon.HWConfig             `mapstructure:",squash"`
	utmcommon.UtmVersionConfig     `mapstructure:",squash"`
	utmcommon.UtmBundleConfig      `mapstructure:",squash"`
	utmcommon.GuestAdditionsConfig `mapstructure:",squash"`
	utmcommon.NoPauseConfig        `mapstructure:",squash"`
	utmcommon.QemuConfig           `mapstructure:",squash"`

	// Set this to true if you would like to use Hypervisor
	// Defaults to false.
	Hypervisor bool `mapstructure:"hypervisor" required:"false"`

	// Set this to true if you would like to use UEFI firmware to boot with
	// UTM. Defaults to false.
	UEFIBoot bool `mapstructure:"uefi_boot" required:"false"`

	// Set this to true if you would like to use local time for base clock
	// Defaults to false.
	// TODO: This is not supported in UTM
	RTCLocalTime bool `mapstructure:"rtc_local_time" required:"false"`

	// The size, in megabytes, of the hard disk to create for the VM. By
	// default, this is 40000 (about 40 GB).
	DiskSize uint `mapstructure:"disk_size" required:"false"`
	// The type of controller that the primary hard drive is attached to,
	// defaults to VirtIO. When set to usb, the drive is attached to an USB
	// controller. When set to scsi, the drive is attached to an  SCSI
	// controller. When set to nvme, the drive is attached to an NVMe
	// controller. When set to virtio, the drive is attached to a VirtIO
	// controller. Please note that when you use "nvme",
	// and you may need to enable EFI mode for nvme to work (this note is from VirtualBox)
	HardDriveInterface string `mapstructure:"hard_drive_interface" required:"false"`
	// The type of controller that the ISO is attached to, defaults to usb.
	// When set to nvme, the drive is attached to an NVMe controller.
	// When set to virtio, the drive is attached to a VirtIO controller.
	ISOInterface string `mapstructure:"iso_interface" required:"false"`
	// Additional disks to create. Attachment starts at 1 since 0
	// is the default disk. Each value represents the disk image size in MiB.
	// Each additional disk uses the same disk parameters as the default disk.
	// Unset by default.
	AdditionalDiskSize []uint `mapstructure:"disk_additional_size" required:"false"`

	// Wheather to resize the cloud image to the disk size. Defaults to false.
	// If set to true, the cloud image will be resized to the disk size.
	// Required qemu-img to be installed in the system.
	ResizeCloudImage bool `mapstructure:"resize_cloud_image" required:"false"`

	// Pass cloud-init data to the VM using a CD-ROM. Defaults to false.
	// If set to true, you must provide cd_files with the path to the cloud-init
	// data and cd_label with value "cidata".
	// If set to false, you must provide http_directory with the cloud-init data.
	UseCD bool `mapstructure:"use_cd" required:"false"`

	// Set this to true if you would like to keep the VM registered with
	// UTM. Defaults to false.
	KeepRegistered bool `mapstructure:"keep_registered" required:"false"`
	// Defaults to false. When enabled, Packer will not export the VM. Useful
	// if the build output is not the resultant image, but created inside the
	// VM.
	SkipExport bool `mapstructure:"skip_export" required:"false"`
	// UTM VM icon.
	VMIcon string `mapstructure:"vm_icon" required:"false"`
	// QEMU system architecture of the virtual machine.
	// For a QEMU virtual machine, you must specify the architecture
	// Which is required in confirguration. By default, this is aarch64.
	// You should use same architecture as the cloud image.
	VMArch string `mapstructure:"vm_arch" required:"false"`
	// Backend to use for the virtual machine.
	// Only qemu cloud images are supported.
	// By default, this is qemu.
	VMBackend string `mapstructure:"vm_backend" required:"false"`
	// This is the name of the utm file for the new virtual machine, without
	// the file extension. By default this is packer-BUILDNAME, where
	// "BUILDNAME" is the name of the build.
	VMName string `mapstructure:"vm_name" required:"false"`

	ctx interpolate.Context
}

func (c *Config) Prepare(raws ...interface{}) ([]string, error) {
	err := config.Decode(c, &config.DecodeOpts{
		PluginType:         BuilderId,
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"guest_additions_path",
				"guest_additions_url",
			},
		},
	}, raws...)
	if err != nil {
		return nil, err
	}

	// Accumulate any errors and warnings
	var errs *packersdk.MultiError
	warnings := make([]string, 0)

	isoWarnings, isoErrs := c.ISOConfig.Prepare(&c.ctx)
	warnings = append(warnings, isoWarnings...)
	errs = packersdk.MultiErrorAppend(errs, isoErrs...)

	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.CDConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(
		errs, c.OutputConfig.Prepare(&c.ctx, &c.PackerConfig)...)
	errs = packersdk.MultiErrorAppend(errs, c.HTTPConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.CommConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.UtmBundleConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.UtmVersionConfig.Prepare(c.Comm.Type)...)
	errs = packersdk.MultiErrorAppend(errs, c.NoPauseConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.QemuConfig.Prepare(&c.ctx)...)

	if c.DiskSize == 0 {
		c.DiskSize = 40960
	}

	if c.HardDriveInterface == "" {
		c.HardDriveInterface = "virtio"
	}

	if c.VMArch == "" {
		c.VMArch = "aarch64"
	}

	if c.VMBackend == "" {
		c.VMBackend = "qemu"
	}
	// Validate and use Enums for the VM backend
	// Only qemu cloud images are supported.
	switch c.VMBackend {
	case "qemu":
		c.VMBackend = "QeMu"
	default:
		errs = packersdk.MultiErrorAppend(
			errs, errors.New("vm_backend must be 'qemu'"))
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf(
			"packer-%s-%d", c.PackerBuildName, interpolate.InitTime.Unix())
	}

	// Validates the presence of the cloud-init data
	// We either use a CD-ROM or HTTP to pass cloud-init data
	if c.UseCD {
		if c.CDFiles == nil && c.CDContent == nil {
			errs = packersdk.MultiErrorAppend(
				errs, errors.New("use_cd is true, but neither cd_files nor cd_content is set"))
		}
		if c.CDLabel != "cidata" {
			errs = packersdk.MultiErrorAppend(
				errs, errors.New("use_cd is true, but cd_label is not set to 'cidata'"))
		}
	} else {
		if c.HTTPDir == "" && c.HTTPContent == nil {
			errs = packersdk.MultiErrorAppend(
				errs, errors.New("use_cd is false, but neither http_directory nor http_content is set"))
		}
	}

	if c.ISOInterface == "" {
		// Default to virtio, In Cloud builder ISO is the primary disk
		c.ISOInterface = "virtio"
	}

	if c.GuestAdditionsInterface == "" {
		c.GuestAdditionsInterface = c.ISOInterface
	}

	switch c.HardDriveInterface {
	case "none", "ide", "scsi", "virtio", "nvme", "usb":
		// do nothing
	default:
		errs = packersdk.MultiErrorAppend(
			errs, errors.New("hard_drive_interface can only be none, ide, scsi, virtio, nvme or usb"))
	}

	switch c.ISOInterface {
	case "ide", "sd", "floppy", "virtio", "nvme", "usb":
		// do nothing
	default:
		errs = packersdk.MultiErrorAppend(
			errs, errors.New("iso_interface can only be ide, sd, floppy, virtio, nvme or usb"))
	}

	// Warnings
	if c.ShutdownCommand == "" {
		warnings = append(warnings,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil

}
