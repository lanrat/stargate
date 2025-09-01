package main

import (
	"context"
	"net"
	"net/netip"
	"strconv"

	"github.com/lanrat/go-socks5"
	"github.com/lanrat/stargate"
)

// runRandomSubnetProxy starts a SOCKS5 proxy server listening on listenAddr that distributes
// connections across random subnets within the specified IP range. It divides the main CIDR
// into smaller subnets of size newCidr and randomly selects a subnet for each connection.
// This is memory efficient for large IPv6 ranges as it doesn't pre-generate all addresses.
// The function cycles through all available subnets before repeating.
// Supports both TCP and UDP protocols simultaneously.
func runRandomSubnetProxy(ctx context.Context, listenAddr string, parsedNetwork netip.Prefix, cidrSize uint) error {
	ipItr, err := stargate.NewRandomIPIterator(parsedNetwork, cidrSize)
	if err != nil {
		return err
	}

	// Parse listen address to get host and port for UDP binding
	host, portStr, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}

	udpDial := func(ctx context.Context, network string, udpClientSrcAddr, targetUDPAddr *net.UDPAddr) (net.Conn, error) {
		return ipItr.Dial(ctx, network, targetUDPAddr.String())
	}

	conf := &socks5.Config{
		Logger:   l,
		Resolver: NewDNSResolver(getCIDRNetwork(parsedNetwork)),
		Dial:     ipItr.Dial,
		DialUDP:  udpDial,
		BindIP:   net.ParseIP(host),
		BindPort: port,
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	// Start TCP SOCKS5 proxy (UDP is handled internally by the server when BindPort is set)
	l.Printf("Starting SOCKS5 proxy on %s", listenAddr)
	return server.ListenAndServe(ctx, listenAddr)
}

// DNSResolver implements socks5.NameResolver using the system DNS resolver.
// It ensures that domain names are resolved to the same IP family (IPv4 or IPv6)
// as the proxy's egress IP.
type DNSResolver struct {
	network  string
	resolver net.Resolver
}

func NewDNSResolver(network string) *DNSResolver {
	d := &DNSResolver{
		network: network,
	}
	return d
}

// Resolve resolves a domain name to an IP address using the system DNS resolver.
// It ensures the resolved IP is in the same address family (IPv4 or IPv6) as specified
// by the network field, which helps maintain consistency with the proxy's egress IP.
func (d *DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	addrs, err := d.resolver.LookupIPAddr(ctx, name)
	if err != nil {
		return ctx, nil, err
	}

	// Filter addresses by the desired IP family
	for _, addr := range addrs {
		if d.network == "ip4" && addr.IP.To4() != nil {
			v("resolved %q to %q", name, addr.IP.String())
			return ctx, addr.IP, nil
		}
		if d.network == "ip6" && addr.IP.To4() == nil && addr.IP.To16() != nil {
			v("resolved %q to %q", name, addr.IP.String())
			return ctx, addr.IP, nil
		}
	}

	return ctx, nil, &net.AddrError{Err: "no suitable address found", Addr: name}
}

// getCIDRNetwork returns "ip4" for IPv4 addresses or "ip6" for IPv6 addresses.
// This is used for DNS resolution context.
func getCIDRNetwork(prefix netip.Prefix) string {
	if prefix.Addr().Is4() {
		return "ip4"
	}
	return "ip6"
}
