package stargate

import (
	"fmt"
	"net"
	"net/netip"
)

// broadcastAddrs is a global map that tracks IP addresses identified as broadcast addresses.
// It is populated by CheckHostConflicts and used to prevent binding to these addresses.
var broadcastAddrs = make(map[string]bool)

// CheckHostConflicts detects if any of the addresses we are going to use are broadcast addresses.
// It populates the global broadcastAddrs map by examining all system network interfaces.
// It returns a list of all conflicting IPs
func CheckHostConflicts(prefix *netip.Prefix) ([]net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	conflictIPs := make([]net.IP, 0)
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			// We are only interested in IPv4 addresses for broadcast.
			if ipnet.IP.To4() == nil {
				continue
			}
			brdIP, err := getBroadcastAddressFromAddr(ipnet)
			if err != nil {
				return nil, err
			}
			brdAddr, ok := netip.AddrFromSlice(brdIP)
			if !ok {
				return nil, fmt.Errorf("unable to parse IP to addr: %+v", brdAddr)
			}
			if prefix.Contains(brdAddr) {
				broadcastAddrs[brdAddr.String()] = true
				v("WARNING: interface %s broadcast address is within provided prefix %s", i.Name, brdIP)
				conflictIPs = append(conflictIPs, brdIP)
			}
		}
	}

	return conflictIPs, nil
}

// getBroadcastAddressFromAddr calculates the broadcast address from a net.IPNet.
// It only supports IPv4 addresses and returns an error for IPv6 or invalid inputs.
func getBroadcastAddressFromAddr(addr net.Addr) (net.IP, error) {
	// Type assertion to check if the net.Addr is a *net.IPNet.
	ipnet, ok := addr.(*net.IPNet)
	if !ok {
		return nil, fmt.Errorf("address is not a net.IPNet type: %T", addr)
	}

	// Check if the IP is an IPv4 address.
	if ipnet.IP.To4() == nil {
		return nil, fmt.Errorf("only IPv4 addresses are supported for broadcast calculation")
	}

	// Perform the bitwise OR calculation.
	// Use IPv4 representation to avoid length mismatches
	ip4 := ipnet.IP.To4()
	mask4 := ipnet.Mask
	if len(mask4) != 4 {
		return nil, fmt.Errorf("invalid IPv4 mask length: %d", len(mask4))
	}

	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = ip4[i] | ^mask4[i]
	}

	return broadcast, nil
}
