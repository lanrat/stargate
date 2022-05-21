package main

import (
	"math"
	"math/big"
	"math/rand"
	"net"
)

// possible enhancement
// dial from iface: https://gist.github.com/creack/43ee6542ddc6fe0da8c02bd723d5cc53

// from: https://gist.github.com/kotakanbe/d3059af990252ba89a82
func hosts(cidr *net.IPNet) ([]net.IP, error) {
	ips := make([]net.IP, 0, maskSize64(&cidr.Mask))
	for ip := cidr.IP.Mask(cidr.Mask); cidr.Contains(ip); inc(ip) {
		// don't add IPv4 addresses ending in .0, on most hosts they leak the real IP
		if ipv4 := ip.To4(); ipv4 != nil && ipv4[3] == 0 {
			continue
		}
		// using dupIP to prevent all of the IP's referencing the same array in memory
		ips = append(ips, dupIP(ip))
	}
	// remove ipv4 broadcast address
	if ip4 := cidr.IP.To4(); ip4 != nil && len(ips) > 1 {
		return ips[0 : len(ips)-1], nil
	}

	return ips, nil
}

// dupIP returns a copy of the provided IP address
func dupIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

//  inc increments an IP
// http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// maskSize64 returns -1 if too large for int64
func maskSize64(m *net.IPMask) int64 {
	maskBits, totalBits := m.Size()
	addrBits := totalBits - maskBits
	if addrBits > 63 { // 63 is max positive int size
		return -1
	}
	return 1 << addrBits
}

// maskSize returns the number of addresses in m
func maskSize(m *net.IPMask) big.Int {
	var size big.Int
	maskBits, totalBits := m.Size()
	addrBits := totalBits - maskBits
	size.Lsh(big.NewInt(1), uint(addrBits))
	return size
}

// randomIP returns a random IP address within the IPNet
func randomIP(cidr *net.IPNet) net.IP {
	ip := cidr.IP
	for i := range ip {
		rb := byte(rand.Intn(math.MaxUint8))
		ip[i] = (cidr.Mask[i] & ip[i]) + (^cidr.Mask[i] & rb)
	}
	return ip
}

// getIPNetwork returns the network string for the IP provided
func getIPNetwork(ip *net.IP) string {
	if ip.To4() != nil {
		return "ip4"
	}
	return "ip6"
}
