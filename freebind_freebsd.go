//go:build freebsd
// +build freebsd

package main

import (
	"fmt"
	"syscall"
)

func controlFreebind(network, address string, c syscall.RawConn) error {
	if err := freeBind(network, address, c); err != nil {
		return err
	}
	return nil
}

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
