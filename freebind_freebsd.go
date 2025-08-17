//go:build freebsd
// +build freebsd

package main

import (
	"fmt"
	"syscall"
)

// controlFreebind sets the IP_BINDANY or IPV6_BINDANY socket option on FreeBSD,
// allowing the socket to bind to IP addresses that are not yet configured on the system.
func controlFreebind(network, address string, c syscall.RawConn) error {
	if err := freeBind(network, address, c); err != nil {
		return err
	}
	return nil
}

// freeBind enables the appropriate BINDANY socket option based on the network type.
// For IPv4 it sets IP_BINDANY, and for IPv6 it sets IPV6_BINDANY.
// from https://github.com/zrepl/zrepl/blob/master/util/tcpsock/tcpsock_freebind_freebsd.go
func freeBind(network, _ string, c syscall.RawConn) error {
	var err, sockErr error
	err = c.Control(func(fd uintptr) {
		switch network {
		case "tcp6":
			sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_BINDANY, 1)
		case "tcp4":
			sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_BINDANY, 1)
		default:
			sockErr = fmt.Errorf("expecting 'tcp6' or 'tcp4', got %q", network)
		}
	})
	if err != nil {
		return err
	}
	return sockErr
}
