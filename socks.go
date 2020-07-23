package main

import (
	"context"
	"net"

	"github.com/haxii/socks5"
)

func runProxy(proxyAddr net.Addr, listenAddr string) error {
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
