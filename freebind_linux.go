//go:build linux

package main

import "syscall"

// controlFreebind sets the IP_FREEBIND socket option on Linux, allowing the socket
// to bind to IP addresses that are not yet configured on the system.
func controlFreebind(network, address string, c syscall.RawConn) error {
	if err := freeBind(network, address, c); err != nil {
		return err
	}
	return nil
}

// freeBind enables the IP_FREEBIND socket option, which allows binding to non-local
// IP addresses. This is essential for egressing traffic from IPs within a routed subnet.
// from https://github.com/zrepl/zrepl/blob/master/util/tcpsock/tcpsock_freebind_linux.go
func freeBind(_, _ string, c syscall.RawConn) error {
	var err, sockErr error
	err = c.Control(func(fd uintptr) {
		// this works for both IPv4 and IPv6
		sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_FREEBIND, 1)
	})
	if err != nil {
		return err
	}
	return sockErr
}
