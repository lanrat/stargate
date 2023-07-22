package main

import (
	"context"
	"net"
	"math/rand"
	"time"
	"github.com/haxii/socks5"
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
// in the cidr range defined. It will shuffle the ranges and then evenly distribute the connections so
// that a subnet is only reused after all the others have been used.
func runRandomSubnetProxy(listenAddr string, iprange string, newCidr uint) error {
	subnets, err := breakIntoSubnets(iprange, newCidr)
    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(subnets), func(i, j int) { subnets[i], subnets[j] = subnets[j], subnets[i] })

	v("we were given network %s with a cidr of %d. our subnet pool size is %d", iprange, newCidr, len(subnets))

	// Keeps track of what subnet we're on in our rotation
	subnet_pointer := 0

	conf := &socks5.Config{
		Logger:   l,
		Resolver: resolver,
	}
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		current_subnet := subnets[subnet_pointer]

		v("pulling ip from subnet %s (range %d/%d)", current_subnet, subnet_pointer, len(subnets))
		_, cidr, _ := net.ParseCIDR(current_subnet)
		ip := randomIP(cidr)
		subnet_pointer++

        // Reset pointer to wrap around once an IP
        // from each subnet has been used.
        if subnet_pointer >= len(subnets) {
            v("used all the subnets in our pool, looping back around...")
            subnet_pointer = 0
        }

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
