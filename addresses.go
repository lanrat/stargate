package main

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/netip"
)

// TODO possible enhancement
// dial from iface: https://gist.github.com/creack/43ee6542ddc6fe0da8c02bd723d5cc53

// hosts returns a slice of all usable host IP addresses within the given CIDR range.
// It excludes the network address and broadcast address for IPv4 networks.
// Uses the modern netip package for cleaner iteration.
// Updated based on: https://gist.github.com/kotakanbe/d3059af990252ba89a82?permalink_comment_id=4105265#gistcomment-4105265
func hosts(cidr *net.IPNet) ([]net.IP, error) {
	// Convert to netip.Prefix for cleaner iteration
	ip, ok := netip.AddrFromSlice(cidr.IP)
	if !ok {
		return nil, fmt.Errorf("invalid IP address")
	}
	ones, _ := cidr.Mask.Size()
	prefix := netip.PrefixFrom(ip.Unmap(), ones)

	var ips []net.IP
	for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
		// Convert netip.Addr back to net.IP
		ips = append(ips, net.IP(addr.AsSlice()))
	}

	// For single host or empty range, return as-is
	if len(ips) < 2 {
		return ips, nil
	}

	// Remove network and broadcast addresses (first and last)
	// This handles both IPv4 and IPv6 appropriately
	return ips[1 : len(ips)-1], nil
}

// maskSize returns the number of addresses in the network mask as a big.Int,
// which can handle arbitrarily large address spaces (e.g., IPv6).
func maskSize(m *net.IPMask) big.Int {
	var size big.Int
	maskBits, totalBits := m.Size()
	addrBits := totalBits - maskBits
	size.Lsh(big.NewInt(1), uint(addrBits))
	return size
}

// randomIP generates a random IP address within the given CIDR range.
// It preserves the network portion and randomizes the host portion.
func randomIP(cidr *net.IPNet) net.IP {
	ip := cidr.IP
	for i := range ip {
		rb := byte(rand.Intn(math.MaxUint8))
		ip[i] = (cidr.Mask[i] & ip[i]) + (^cidr.Mask[i] & rb)
	}
	return ip
}

// getIPNetwork returns "ip4" for IPv4 addresses or "ip6" for IPv6 addresses.
// This is used for DNS resolution context.
func getIPNetwork(ip *net.IP) string {
	if ip.To4() != nil {
		return "ip4"
	}
	return "ip6"
}
