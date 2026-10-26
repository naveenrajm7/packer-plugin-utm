package ipsw

import (
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	utmcommon "github.com/naveenrajm7/packer-plugin-utm/builder/utm/common"
)

// macOSSSHHost returns an explicit override or asks UTM for the guest IP.
// StepConnect retries this callback while the VM boots and obtains DHCP.
func macOSSSHHost(override string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		if override != "" {
			return override, nil
		}
		vmID := state.Get("vmId").(string)
		output, err := state.Get("driver").(utmcommon.Driver).Utmctl("ip-address", vmID)
		if err != nil {
			return "", fmt.Errorf("querying IP address for %s: %w", vmID, err)
		}
		var ipv6 string
		for _, line := range strings.Split(output, "\n") {
			address := strings.TrimSpace(line)
			ip := net.ParseIP(address)
			if ip == nil || !ip.IsGlobalUnicast() {
				continue
			}
			if ip.To4() != nil {
				return address, nil
			}
			if ipv6 == "" {
				ipv6 = address
			}
		}
		if ipv6 != "" {
			return ipv6, nil
		}
		return "", fmt.Errorf("UTM has not reported a usable guest IP address yet")
	}
}
