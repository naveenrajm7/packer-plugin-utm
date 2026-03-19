// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"testing"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestArtifact_ImplementsArtifact(t *testing.T) {
	var _ packersdk.Artifact = &Artifact{}
}

func TestArtifact_Id(t *testing.T) {
	artifact := NewArtifact("vmware", "./")
	if artifact.Id() != "vmware" {
		t.Fatalf("should return name as Id")
	}
}
