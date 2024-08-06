//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package zip

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// Fields from config file
	OutputPath string `mapstructure:"output"`

	// Derived fields
	Archive   string
	Algorithm string

	ctx interpolate.Context
}

// PostProcessor implements packersdk.PostProcessor
// Creates a zip archive of a given UTM directory (UTM VM bundle)
type PostProcessor struct {
	config Config
}

func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec { return p.config.FlatMapstructure().HCL2Spec() }

func (p *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		PluginType:         "zip",
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{"output"},
		},
	}, raws...)
	if err != nil {
		return err
	}

	errs := new(packersdk.MultiError)

	// If there is no explicit number of Go threads to use, then set it
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	if p.config.OutputPath == "" {
		p.config.OutputPath = "packer_{{.BuildName}}_{{.BuilderType}}"
	}

	if err = interpolate.Validate(p.config.OutputPath, &p.config.ctx); err != nil {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf("Error parsing target template: %s", err))
	}

	if len(errs.Errors) > 0 {
		return errs
	}

	return nil
}

func (p *PostProcessor) PostProcess(
	ctx context.Context,
	ui packersdk.Ui,
	artifact packersdk.Artifact,
) (packersdk.Artifact, bool, bool, error) {
	var generatedData map[interface{}]interface{}
	stateData := artifact.State("generated_data")
	if stateData != nil {
		// Make sure it's not a nil map so we can assign to it later.
		generatedData = stateData.(map[interface{}]interface{})
	}
	// If stateData has a nil map generatedData will be nil
	// and we need to make sure it's not
	if generatedData == nil {
		generatedData = make(map[interface{}]interface{})
	}

	// These are extra variables that will be made available for interpolation.
	generatedData["BuildName"] = p.config.PackerBuildName
	generatedData["BuilderType"] = p.config.PackerBuilderType
	p.config.ctx.Data = generatedData

	target, err := interpolate.Render(p.config.OutputPath, &p.config.ctx)
	if err != nil {
		return nil, false, false, fmt.Errorf("Error interpolating output value: %s", err)
	} else {
		fmt.Println(target)
	}

	newArtifact := &Artifact{Path: target}

	ui.Say(fmt.Sprintf("Zipping %s", target))

	// Find path to UTM directory in our artifact
	utmDir, err := findUTMDirectory(artifact)
	// Pass the directory of artifact to create zip archive
	err = zipDirectory(utmDir, target)

	if err != nil {
		return nil, false, false, fmt.Errorf("Error creating zip: %s", err)
	}

	ui.Say(fmt.Sprintf("Archive %s completed", target))

	return newArtifact, false, false, nil
}

func findUTMDirectory(artifact packersdk.Artifact) (string, error) {
	for _, file := range artifact.Files() {
		if idx := strings.Index(file, ".utm/"); idx != -1 {
			return file[:idx+4], nil // +4 to include ".utm"
		}
	}
	return "", errors.New("no .utm directory found")
}

func zipDirectory(sourceDir, zipFile string) error {
	// Create a new zip file
	zipfile, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	// Create a new zip writer
	zipWriter := zip.NewWriter(zipfile)
	defer zipWriter.Close()

	// Walk through the source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set the header name to the relative path
		header.Name, err = filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return err
		}

		// If the file is a directory, add a trailing slash to the header name
		if info.IsDir() {
			header.Name += "/"
		} else {
			// Set the method to deflate for files
			header.Method = zip.Deflate
		}

		// Create a writer for the file in the zip archive
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// If the file is not a directory, copy its contents to the zip writer
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
