// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func CommHost(host string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		return host, nil
	}
}

func CommPort(state multistep.StateBag) (int, error) {
	commHostPort := state.Get("commHostPort").(int)
	return commHostPort, nil
}
