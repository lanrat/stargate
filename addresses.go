package main

import (
	"net"
)

// from: https://gist.github.com/kotakanbe/d3059af990252ba89a82

func hosts(cidr string) ([]string, error) {
	ip, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(net.Mask); net.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// // remove broadcast address
	// if len(ips) > 1 {
	// 	return ips[0 : len(ips)-1], nil
	// }
	return ips, nil
}

func ips2Address(ips []string) ([]net.Addr, error) {
	var address net.Addr
	var err error
	addresses := make([]net.Addr, 0, len(ips))
	for _, ip := range ips {
		if *udp {
			address, err = net.ResolveUDPAddr("udp", net.JoinHostPort(ip, "0"))
		} else {
			address, err = net.ResolveTCPAddr("tcp", net.JoinHostPort(ip, "0"))
		}
		if err != nil {
			return addresses, err
		}
		addresses = append(addresses, address)
	}
	return addresses, nil
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

// possible enhancement
// dial from iface: https://gist.github.com/creack/43ee6542ddc6fe0da8c02bd723d5cc53
