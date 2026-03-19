// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"archive/tar"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/tmp"
)

var (
	// ErrInvalidCompressionLevel is returned when the compression level passed
	// to gzip is not in the expected range. See compress/flate for details.
	ErrInvalidCompressionLevel = fmt.Errorf(
		"invalid compression level. Expected an integer from -1 to 9")
)

// Copies a file by copying the contents of the file to another place.
func CopyContents(dst, src string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcF.Close() }()

	dstDir, _ := filepath.Split(dst)
	if dstDir != "" {
		err := os.MkdirAll(dstDir, 0755)
		if err != nil {
			return err
		}
	}

	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstF.Close() }()

	if _, err := io.Copy(dstF, srcF); err != nil {
		return err
	}

	return nil
}

// Copies the contents of a directory to another place.
func CopyDirectoryContents(dst, src string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return CopyContents(dstPath, path)
	})
}

// Creates a (hard) link to a file, ensuring that all parent directories also exist.
func LinkFile(dst, src string) error {
	dstDir, _ := filepath.Split(dst)
	if dstDir != "" {
		err := os.MkdirAll(dstDir, 0755)
		if err != nil {
			return err
		}
	}

	if err := os.Link(src, dst); err != nil {
		return err
	}

	return nil
}

// DirToBox takes the directory and compresses it into a Vagrant-compatible
// box. This function does not perform checks to verify that dir is
// actually a proper box. This is an expected precondition.
func DirToBox(dst, dir string, ui packersdk.Ui, level int) error {
	log.Printf("Turning dir into box: %s => %s", dir, dst)

	// Make the containing directory, if it does not already exist
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}

	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstF.Close() }()

	var dstWriter io.WriteCloser = dstF
	if level != flate.NoCompression {
		log.Printf("Compressing with gzip compression level: %d", level)
		gzipWriter, err := makePgzipWriter(dstWriter, level)
		if err != nil {
			return err
		}
		defer func() { _ = gzipWriter.Close() }()

		dstWriter = gzipWriter
	}

	tarWriter := tar.NewWriter(dstWriter)
	defer func() { _ = tarWriter.Close() }()

	// This is the walk func that tars each of the files in the dir
	tarWalk := func(path string, info os.FileInfo, prevErr error) error {
		// If there was a prior error, return it
		if prevErr != nil {
			return prevErr
		}

		// Skip directories
		if info.IsDir() {
			log.Printf("Skipping directory '%s' for box '%s'", path, dst)
			return nil
		}

		log.Printf("Box add: '%s' to '%s'", path, dst)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// go >=1.10 wants to use GNU tar format to workaround issues in
		// libarchive < 3.3.2
		setHeaderFormat(header)

		// We have to set the Name explicitly because it is supposed to
		// be a relative path to the root. Otherwise, the tar ends up
		// being a bunch of files in the root, even if they're actually
		// nested in a dir in the original "dir" param.
		header.Name, err = filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if ui != nil {
			ui.Say(fmt.Sprintf("Compressing: %s", header.Name))
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}

		return nil
	}

	// Tar.gz everything up
	return filepath.Walk(dir, tarWalk)
}

// CreateDummyBox create a dummy Vagrant-compatible box under temporary dir
// This function is mainly used to check cases such as the host system having
// a GNU tar incompatible uname that will cause the actual Vagrant box creation
// to fail later
func CreateDummyBox(ui packersdk.Ui, level int) error {
	ui.Say("Creating a dummy Vagrant box to ensure the host system can create one correctly")

	// Create a temporary dir to create dummy Vagrant box from
	tempDir, err := tmp.Dir("packer")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Write some dummy metadata for the box
	if err := WriteMetadata(tempDir, make(map[string]string)); err != nil {
		return err
	}

	// Create the dummy Vagrant box
	tempBox, err := tmp.File("box-*.box")
	if err != nil {
		return err
	}
	defer func() { _ = tempBox.Close() }()
	defer func() { _ = os.Remove(tempBox.Name()) }()
	if err := DirToBox(tempBox.Name(), tempDir, nil, level); err != nil {
		return err
	}

	return nil
}

// WriteMetadata writes the "metadata.json" file for a Vagrant box.
func WriteMetadata(dir string, contents interface{}) error {
	if _, err := os.Stat(filepath.Join(dir, "metadata.json")); os.IsNotExist(err) {
		f, err := os.Create(filepath.Join(dir, "metadata.json"))
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		enc := json.NewEncoder(f)
		return enc.Encode(contents)
	}

	return nil
}

func makePgzipWriter(output io.WriteCloser, compressionLevel int) (io.WriteCloser, error) {
	// Use standard gzip instead of pgzip to avoid corruption with large files
	// pgzip parallel compression can have issues with very large QCOW2 files (>1GB)
	gzipWriter, err := gzip.NewWriterLevel(output, compressionLevel)
	if err != nil {
		return nil, ErrInvalidCompressionLevel
	}
	return gzipWriter, nil
}
