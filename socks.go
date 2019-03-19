package main

import (
	"context"
	"fmt"
	"net"

	"github.com/haxii/socks5"
)

func runProxy(proxyIP, listenAddr string) error {
	proxyAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", proxyIP))
	if err != nil {
		return err
	}
	conf := &socks5.Config{}
	conf.Logger = l
	conf.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := net.Dialer{LocalAddr: proxyAddr}
		return d.DialContext(ctx, network, addr)
	}
	server, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return server.ListenAndServe("tcp", listenAddr)
}
