package ipsw

import "testing"

func TestConfigModes(t *testing.T) {
	base := map[string]interface{}{
		"ipsw_url": "file:///tmp/restore.ipsw", "ipsw_checksum": "none",
	}
	var artifact Config
	if _, err := artifact.Prepare(base); err != nil {
		t.Fatalf("artifact config: %v", err)
	}
	if artifact.Type != "none" || artifact.CPUs != 4 || artifact.Memory != 8192 {
		t.Fatalf("unexpected artifact defaults: %#v", artifact)
	}

	var incompleteSSH Config
	withSSH := map[string]interface{}{
		"ipsw_url": "file:///tmp/restore.ipsw", "ipsw_checksum": "none", "communicator": "ssh",
	}
	if _, err := incompleteSSH.Prepare(withSSH); err == nil {
		t.Fatal("expected SSH provisioning credentials and host to be required")
	}
	withSSH["ssh_username"] = "packer"
	withSSH["ssh_password"] = "secret"
	var completeSSH Config
	if _, err := completeSSH.Prepare(withSSH); err != nil {
		t.Fatalf("SSH config: %v", err)
	}
}

func TestSupportsIPSW(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    bool
	}{
		{"4.7.5", false},
		{"5.0.5", false},
		{"5.0.6", true},
		{"5.0.8", true},
		{"5.1.0", true},
		{"6.0.0", true},
		{"invalid", false},
	} {
		if got := supportsIPSW(tc.version); got != tc.want {
			t.Errorf("supportsIPSW(%q) = %v, want %v", tc.version, got, tc.want)
		}
	}
}
