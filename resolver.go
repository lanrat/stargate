package main

import (
	"context"
	"net"
)

// DNSResolver implements socks5.NameResolver using the system DNS resolver.
// It ensures that domain names are resolved to the same IP family (IPv4 or IPv6)
// as the proxy's egress IP.
type DNSResolver struct {
	network string
}

// Resolve resolves a domain name to an IP address using the system DNS resolver.
// It ensures the resolved IP is in the same address family (IPv4 or IPv6) as specified
// by the network field, which helps maintain consistency with the proxy's egress IP.
// TODO use context for name resolution
func (d DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	//v("resolving %q: %q", d.network, name)
	addr, err := net.ResolveIPAddr(d.network, name)
	if err != nil {
		return ctx, nil, err
	}
	v("resolved %q to %q", name, addr.IP.String())
	return ctx, addr.IP, err
}
