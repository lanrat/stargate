package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/haxii/socks5"
	//"golang.zx2c4.com/wireguard/tun/netstack"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// runProxy starts a SOCKS proxy for proxyIP listening on listenAddr
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

func runWgProxy(cidr *net.IPNet, listenAddr string, tnet *stack.Net) error {
	conf := &socks5.Config{
		Logger:   l,
		Resolver: resolver,
	}
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		ip := randomIP(cidr)
		v("%s wg proxy request for: %q, using %s", network, addr, ip)
		localAddr, _ := netip.AddrFromSlice(ip)
		localAddrPort := netip.AddrPortFrom(localAddr, 0) // using port 0 lets the OS decide
		remoteAddrPort, err := netip.ParseAddrPort(addr)
		if err != nil {
			return nil, fmt.Errorf("unable to parse netip.AddrPort from %q", addr)
		}
		if strings.HasPrefix(network, "tcp") {
			return tnet.DialTCPWithBindAddr(ctx, localAddrPort, remoteAddrPort)
		} else if strings.HasPrefix(network, "udp") {
			return tnet.DialUDPAddrPort(localAddrPort, remoteAddrPort)
		} else if strings.HasPrefix(network, "ping") {
			return tnet.DialPingAddr(localAddr, remoteAddrPort.Addr())
		}

		return nil, fmt.Errorf("cant use network %q, non tcp not implemented yet", network)

	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe("tcp", listenAddr)
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
