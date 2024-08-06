// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utm

import (
	"io/ioutil"
	"os"
	"testing"
)

func testConfig(t *testing.T) map[string]interface{} {
	return map[string]interface{}{
		"ssh_username":     "foo",
		"shutdown_command": "foo",
		"source_path":      "config_test.go",
	}
}

func getTempFile(t *testing.T) *os.File {
	tf, err := ioutil.TempFile("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	tf.Close()

	// don't forget to cleanup the file downstream:
	// defer os.Remove(tf.Name())

	return tf
}

func TestNewConfig_sourcePath(t *testing.T) {
	// Okay, because it gets caught during download
	cfg := testConfig(t)
	delete(cfg, "source_path")
	var c Config
	warns, err := c.Prepare(cfg)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatalf("should error with empty `source_path`")
	}

	// Good
	tf := getTempFile(t)
	defer os.Remove(tf.Name())

	cfg = testConfig(t)
	cfg["source_path"] = tf.Name()
	warns, err = c.Prepare(cfg)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestNewConfig_shutdown_timeout(t *testing.T) {
	cfg := testConfig(t)
	tf := getTempFile(t)
	defer os.Remove(tf.Name())

	// Expect this to fail
	cfg["source_path"] = tf.Name()
	cfg["shutdown_timeout"] = "NaN"
	var c Config
	warns, err := c.Prepare(cfg)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should error")
	}

	// Passes when given a valid time duration
	cfg["shutdown_timeout"] = "10s"
	warns, err = c.Prepare(cfg)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
}
