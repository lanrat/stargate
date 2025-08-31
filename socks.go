package main

import (
	"net/netip"

	"github.com/haxii/socks5"
)

// runRandomSubnetProxy starts a SOCKS5 proxy server listening on listenAddr that distributes
// connections across random subnets within the specified IP range. It divides the main CIDR
// into smaller subnets of size newCidr and randomly selects a subnet for each connection.
// This is memory efficient for large IPv6 ranges as it doesn't pre-generate all addresses.
// The function cycles through all available subnets before repeating.
func runRandomSubnetProxy(listenAddr string, parsedNetwork netip.Prefix, cidrSize uint) error {
	ipItr, err := NewRandomIPIterator(parsedNetwork, cidrSize)
	if err != nil {
		return err
	}
	conf := &socks5.Config{
		Logger:   l,
		Resolver: resolver,
		Dial:     ipItr.Dial,
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe("tcp", listenAddr)
}

// getCIDRNetwork returns "ip4" for IPv4 addresses or "ip6" for IPv6 addresses.
// This is used for DNS resolution context.
func getCIDRNetwork(prefix netip.Prefix) string {
	if prefix.Addr().Is4() {
		return "ip4"
	}
	return "ip6"
}
