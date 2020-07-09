package main

import (
	"context"
	"math/rand"
	"net"

	"github.com/haxii/socks5"
)

func runProxy(proxyAddr net.Addr, listenAddr string) error {
	conf := &socks5.Config{}
	conf.Logger = l
	d := net.Dialer{LocalAddr: proxyAddr}
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return d.DialContext(ctx, network, addr)
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe("tcp", listenAddr)
}

func runRandomProxy(proxyAddresses []net.Addr, listenAddr string) error {
	conf := &socks5.Config{}
	conf.Logger = l
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := net.Dialer{LocalAddr: proxyAddresses[rand.Intn(len(proxyAddresses))]}
		return d.DialContext(ctx, network, addr)
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe("tcp", listenAddr)
}
