package main

import (
	"context"
	"math/big"
	"net"
	"net/netip"

	"github.com/haxii/socks5"
	"github.com/lanrat/stargate/permute"
)

// runProxy starts a SOCKS proxy for proxyAddr listening on listenAddr
func runProxy(proxyIP net.IP, listenAddr string) error {
	proxyAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(proxyIP.String(), "0"))
	if err != nil {
		return err
	}
	conf := &socks5.Config{
		Logger:   l,
		Resolver: resolver,
	}
	d := net.Dialer{
		LocalAddr: proxyAddr,
		Control:   controlFreebind,
	}
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		v("%s proxy request for: %q", network, addr)
		return d.DialContext(ctx, network, addr)
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe(proxyAddr.Network(), listenAddr)
}

// runRandomProxy starts a proxy listening on listenAddr that egresses every connection on a new random port in cider
func runRandomProxy(cidr *net.IPNet, listenAddr string) error {
	conf := &socks5.Config{
		Logger:   l,
		Resolver: resolver,
	}
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		ip := randomIP(cidr)
		v("random %s proxy (%q) request for: %q", network, ip.String(), addr)
		d := net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: ip,
			},
			Control: controlFreebind,
		}
		return d.DialContext(ctx, network, addr)
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe("tcp", listenAddr)
}

// runRandomSubnetProxy starts a proxy listening on listenAddr that egresses every connection on a subnet
// in the cidr range defined. It uses a RandomParallelIterator to evenly distribute connections across
// subnets without pre-generating all subnet addresses (memory efficient for large IPv6 ranges).
func runRandomSubnetProxy(listenAddr string, iprange string, newCidr uint) error {
	parsedNetwork, err := netip.ParsePrefix(iprange)
	if err != nil {
		return err
	}

	subnetCountVal := subnetCount(parsedNetwork, int(newCidr))
	if subnetCountVal == 0 {
		return err
	}

	v("we were given network %s with a cidr of %d. our subnet pool size is %d", iprange, newCidr, subnetCountVal)

	// Create a RandomParallelIterator for subnet indices
	iterator, err := permute.NewRandomParallelIterator(big.NewInt(0), new(big.Int).SetUint64(subnetCountVal-1))
	if err != nil {
		return err
	}

	conf := &socks5.Config{
		Logger:   l,
		Resolver: resolver,
	}
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Get next random subnet index
		subnetIndex, ok := iterator.Next()
		if !ok {
			// All subnets have been used, create a new iterator to start over
			iterator, err = permute.NewRandomParallelIterator(big.NewInt(0), new(big.Int).SetUint64(subnetCountVal-1))
			if err != nil {
				return nil, err
			}
			v("used all the subnets in our pool, looping back around...")
			subnetIndex, _ = iterator.Next()
		}

		// Get the subnet at this index
		subnetPrefix, ok := nthSubnet(parsedNetwork, int(newCidr), subnetIndex.Uint64())
		if !ok {
			return nil, err
		}

		// Convert netip.Prefix to net.IPNet for use with randomIP
		ip := subnetPrefix.Addr()
		var subnet *net.IPNet
		if ip.Is4() {
			ipv4 := ip.As4()
			subnet = &net.IPNet{
				IP:   net.IP(ipv4[:]),
				Mask: net.CIDRMask(subnetPrefix.Bits(), 32),
			}
		} else {
			ipv6 := ip.As16()
			subnet = &net.IPNet{
				IP:   net.IP(ipv6[:]),
				Mask: net.CIDRMask(subnetPrefix.Bits(), 128),
			}
		}

		v("pulling ip from subnet %s (index %s/%d)", subnet.String(), subnetIndex.String(), subnetCountVal)
		randIP := randomIP(subnet)

		v("random %s proxy (%q) request for: %q", network, randIP.String(), addr)
		d := net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: randIP,
			},
			Control: controlFreebind,
		}
		return d.DialContext(ctx, network, addr)
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe("tcp", listenAddr)
}
