//go:build !linux && !freebsd
// +build !linux,!freebsd

package main

import "syscall"

// leave nil
var controlFreebind func(network, address string, c syscall.RawConn) error = nil
