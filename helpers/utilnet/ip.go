package utilnet

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var InvalidIP = errors.New("invalid ip format")

func SplitIpAndPort(addr string) (ip string, port string, err error) {
	if addr[0] == '[' {
		i := strings.Index(addr, "]")
		if i == -1 {
			err = InvalidIP
			return
		}
		if i == len(addr)-1 {
			ip = addr[1:i]
			return
		}
		if addr[i+1] == ':' {
			ip = addr[1:i]
			port = addr[i+2:]
			if len(port) == 0 {
				err = InvalidIP
				return
			}
			return
		}
		err = InvalidIP
		return
	}

	colonCount := strings.Count(addr, ":")
	if colonCount == 1 || colonCount == 8 {
		i := strings.LastIndex(addr, ":")
		ip = addr[0:i]
		port = addr[i+1:]
		return
	}
	if colonCount != 0 {
		err = InvalidIP
		return
	}
	ip = addr
	return
}

// AddressesAndRangesToIPNets converts a slice of strings with IPv4 and IPv6 addresses and
// CIDR ranges (prefixes) to net.IPNet instances.
// If net.ParseCIDR or net.ParseIP fail, an error will be returned.
// Zones in addresses or ranges are not allowed and will result in an error. This is because:
// a) net.ParseCIDR will fail to parse a range with a zone, and
// b) netip.ParsePrefix will succeed but silently throw away the zone; then
// netip.Prefix.Contains will return false for any IP with a zone, causing confusion and bugs.
func AddressesAndRangesToIPNets(ranges ...string) ([]net.IPNet, error) {
	var result []net.IPNet
	for _, r := range ranges {
		if strings.Contains(r, "%") {
			return nil, fmt.Errorf("zones are not allowed: %q", r)
		}

		if strings.Contains(r, "/") {
			// This is a CIDR/prefix
			_, ipNet, err := net.ParseCIDR(r)
			if err != nil {
				return nil, fmt.Errorf("net.ParseCIDR failed for %q: %w", r, err)
			}
			result = append(result, *ipNet)
		} else {
			// This is a single IP; convert it to a range including only itself
			ip := net.ParseIP(r)
			if ip == nil {
				return nil, fmt.Errorf("net.ParseIP failed for %q", r)
			}

			// To use the right size IP and  mask, we need to know if the address is IPv4 or v6.
			// Attempt to convert it to IPv4 to find out.
			if ipv4 := ip.To4(); ipv4 != nil {
				ip = ipv4
			}

			// Mask all the bits
			mask := len(ip) * 8
			result = append(result, net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(mask, mask),
			})
		}
	}

	return result, nil
}
