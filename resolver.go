package main

import (
	"context"
	"net"
)

// DNSResolver uses the system DNS to resolve host names
type DNSResolver struct {
	network string
}

// Resolve with but use the same address family as the binding IP
func (d DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	addr, err := net.ResolveIPAddr(d.network, name)
	if err != nil {
		return ctx, nil, err
	}
	return ctx, addr.IP, err
}
