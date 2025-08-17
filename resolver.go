package main

import (
	"context"
	"net"
)

// DNSResolver uses the system DNS to resolve host names
type DNSResolver struct {
	network string
}

// Resolve using same address family as the binding IP
func (d DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	//v("resolving %q: %q", d.network, name)
	addr, err := net.ResolveIPAddr(d.network, name)
	if err != nil {
		return ctx, nil, err
	}
	v("resolved %q to %q", name, addr.IP.String())
	return ctx, addr.IP, err
}
