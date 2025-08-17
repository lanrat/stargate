package main

import (
	"context"
	"math/big"
	"net"

	netaddr "github.com/dspinhirne/netaddr-go"
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
	ip, _, err := net.ParseCIDR(iprange)
	if err != nil {
		return err
	}

	var parsedNetwork interface{}
	var subnetCount uint64
	isIPv4 := ip.To4() != nil

	if isIPv4 {
		parsedNetworkV4, err := netaddr.ParseIPv4Net(iprange)
		if err != nil {
			return err
		}
		parsedNetwork = parsedNetworkV4
		subnetCount = uint64(parsedNetworkV4.SubnetCount(newCidr))
	} else {
		parsedNetworkV6, err := netaddr.ParseIPv6Net(iprange)
		if err != nil {
			return err
		}
		parsedNetwork = parsedNetworkV6
		subnetCount = parsedNetworkV6.SubnetCount(newCidr)
	}

	v("we were given network %s with a cidr of %d. our subnet pool size is %d", iprange, newCidr, subnetCount)

	// Create a RandomParallelIterator for subnet indices
	iterator, err := permute.NewRandomParallelIterator(big.NewInt(0), new(big.Int).SetUint64(subnetCount-1))
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
			iterator, err = permute.NewRandomParallelIterator(big.NewInt(0), new(big.Int).SetUint64(subnetCount-1))
			if err != nil {
				return nil, err
			}
			v("used all the subnets in our pool, looping back around...")
			subnetIndex, _ = iterator.Next()
		}

		// Get the subnet at this index
		var subnet *net.IPNet
		if isIPv4 {
			parsedNetworkV4 := parsedNetwork.(*netaddr.IPv4Net)
			subnetNetaddr := parsedNetworkV4.NthSubnet(newCidr, uint32(subnetIndex.Uint64()))
			if subnetNetaddr != nil {
				_, subnet, _ = net.ParseCIDR(subnetNetaddr.String())
			}
		} else {
			parsedNetworkV6 := parsedNetwork.(*netaddr.IPv6Net)
			subnetNetaddr := parsedNetworkV6.NthSubnet(newCidr, subnetIndex.Uint64())
			if subnetNetaddr != nil {
				_, subnet, _ = net.ParseCIDR(subnetNetaddr.String())
			}
		}

		if subnet == nil {
			return nil, err
		}

		v("pulling ip from subnet %s (index %s/%d)", subnet.String(), subnetIndex.String(), subnetCount)
		ip := randomIP(subnet)

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
