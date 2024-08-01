// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"os"
	"path/filepath"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This is the common builder ID to all of these artifacts.
const BuilderId = "naveenrajm7.utm"

// Artifact is the result of running the UTM builder, namely a directory
// of files associated with the resulting machine.
type artifact struct {
	// The ID of the artifact, which is the name of the VM
	id string
	// The directory containing the VM files (.utm)
	dir string
	// The files in the directory
	f []string

	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

// NewArtifact returns a UTM artifact containing a .utm
// directory (file for UTM, which can be imported into UTM).
// in the given output directory
func NewArtifact(dir string, vmName string, generatedData map[string]interface{}) (packersdk.Artifact, error) {
	files := make([]string, 0, 5)
	visit := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}

		return err
	}

	if err := filepath.Walk(dir, visit); err != nil {
		return nil, err
	}

	return &artifact{
		id:        vmName,
		dir:       dir,
		f:         files,
		StateData: generatedData,
	}, nil
}

func (*artifact) BuilderId() string {
	return BuilderId
}

func (a *artifact) Files() []string {
	return a.f
}

func (a *artifact) Id() string {
	return a.id
}

func (a *artifact) String() string {
	return fmt.Sprintf("VM (.utm file) is in directory : %s", a.dir)
}

func (a *artifact) State(name string) interface{} {
	return a.StateData[name]
}

func (a *artifact) Destroy() error {
	return os.RemoveAll(a.dir)
}
