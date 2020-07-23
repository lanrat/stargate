package main

import "syscall"

func controlFreebind(network, address string, c syscall.RawConn) error {
	if err := freeBind(network, address, c); err != nil {
		return err
	}
	return nil
}

// from https://github.com/zrepl/zrepl/blob/master/util/tcpsock/tcpsock_freebind_linux.go
func freeBind(network, address string, c syscall.RawConn) error {
	var err, sockerr error
	err = c.Control(func(fd uintptr) {
		// apparently, this works for both IPv4 and IPv6
		sockerr = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_FREEBIND, 1)
	})
	if err != nil {
		return err
	}
	return sockerr
}
