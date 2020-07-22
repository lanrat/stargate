package main

import (
	"math"
	"math/rand"
	"net"
)

// possible enhancement
// dial from iface: https://gist.github.com/creack/43ee6542ddc6fe0da8c02bd723d5cc53

// from: https://gist.github.com/kotakanbe/d3059af990252ba89a82
func hosts(cidr *net.IPNet) ([]string, error) {
	ips := make([]string, 0, maskSize(&cidr.Mask))
	for ip := cidr.IP.Mask(cidr.Mask); cidr.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove ipv4 broadcast address
	if ip4 := cidr.IP.To4(); ip4 != nil && len(ips) > 1 {
		return ips[0 : len(ips)-1], nil
	}
	return ips, nil
}

//  http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func ips2Address(ips []string) ([]net.Addr, error) {
	var address net.Addr
	var err error
	addresses := make([]net.Addr, 0, len(ips))
	for _, ip := range ips {
		address, err = net.ResolveTCPAddr("tcp", net.JoinHostPort(ip, "0"))
		if err != nil {
			return addresses, err
		}
		addresses = append(addresses, address)
	}
	return addresses, nil
}

// returns -1 if too large for int64
func maskSize(m *net.IPMask) int64 {
	maskBits, totalBits := m.Size()
	addrBits := totalBits - maskBits
	if addrBits > math.MaxInt64 {
		return -1
	}
	return 1 << addrBits
}
func randomIP(cidr *net.IPNet) net.IP {
	ip := cidr.IP
	for i := range ip {
		ip[i] = cidr.Mask[i] & (ip[i] + byte(rand.Intn(math.MaxUint8-int(ip[i]))))
	}
	return ip
}
