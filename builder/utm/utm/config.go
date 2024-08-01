//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package utm

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/common"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

// Config is the configuration structure for the builder.
type Config struct {
	common.PackerConfig    `mapstructure:",squash"`
	utmcommon.ExportConfig `mapstructure:",squash"`
	utmcommon.OutputConfig `mapstructure:",squash"`
	// TODO: Use run config to fill remote connection details
	// like VRDP for VirtualBox, VNC for UTM (QEMU) ?
	// RunConfig           `mapstructure:",squash"`
	utmcommon.CommConfig       `mapstructure:",squash"`
	utmcommon.ShutdownConfig   `mapstructure:",squash"`
	utmcommon.UtmVersionConfig `mapstructure:",squash"`
	// The checksum for the source_path file. The type of the checksum is
	// specified within the checksum field as a prefix, ex: "md5:{$checksum}".
	// The type of the checksum can also be omitted and Packer will try to
	// infer it based on string length. Valid values are "none", "{$checksum}",
	// "md5:{$checksum}", "sha1:{$checksum}", "sha256:{$checksum}",
	// "sha512:{$checksum}" or "file:{$path}". Here is a list of valid checksum
	// values:
	//  * md5:090992ba9fd140077b0661cb75f7ce13
	//  * 090992ba9fd140077b0661cb75f7ce13
	//  * sha1:ebfb681885ddf1234c18094a45bbeafd91467911
	//  * ebfb681885ddf1234c18094a45bbeafd91467911
	//  * sha256:ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
	//  * ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
	//  * file:http://releases.ubuntu.com/20.04/SHA256SUMS
	//  * file:file://./local/path/file.sum
	//  * file:./local/path/file.sum
	//  * none
	// Although the checksum will not be verified when it is set to "none",
	// this is not recommended since these files can be very large and
	// corruption does happen from time to time.
	Checksum string `mapstructure:"checksum" required:"true"`
	// The filepath or URL to a UTM file that acts as the
	// source of this build.
	SourcePath string `mapstructure:"source_path" required:"true"`
	// The path where the UTM file should be saved
	// after download. By default, it will go in the packer cache, with a hash of
	// the original filename as its name.
	TargetPath string `mapstructure:"target_path" required:"false"`
	// This is the name of the UTM file for the new virtual machine, without
	// the file extension. Make sure VMName in UTM after import is same
	// as the UTM file name, By default this is packer-BUILDNAME,
	// where "BUILDNAME" is the name of the build.
	VMName string `mapstructure:"vm_name" required:"false"`
	// Set this to true if you would like to keep
	// the VM registered with UTM. Defaults to false.
	KeepRegistered bool `mapstructure:"keep_registered" required:"false"`
	// Defaults to false. When enabled, Packer will
	// not export the VM. Useful if the build output is not the resultant image,
	// but created inside the VM.
	SkipExport bool `mapstructure:"skip_export" required:"false"`

	ctx interpolate.Context
}

func (c *Config) Prepare(raws ...interface{}) ([]string, error) {
	err := config.Decode(c, &config.DecodeOpts{
		PluginType:         utmcommon.BuilderId, // "naveenrajm7.utm"
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
			},
		},
	}, raws...)
	if err != nil {
		return nil, err
	}

	// Defaults
	if c.VMName == "" {
		c.VMName = fmt.Sprintf(
			"packer-%s-%d", c.PackerBuildName, interpolate.InitTime.Unix())
	}

	// Prepare the errors
	var errs *packersdk.MultiError
	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.OutputConfig.Prepare(&c.ctx, &c.PackerConfig)...)
	// errs = packersdk.MultiErrorAppend(errs, c.RunConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.CommConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.UtmVersionConfig.Prepare(c.CommConfig.Comm.Type)...)

	if c.SourcePath == "" {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("source_path is required"))
	}

	// Warnings
	var warnings []string
	if c.ShutdownCommand == "" {
		warnings = append(warnings,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	// Check for any errors.
	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}
