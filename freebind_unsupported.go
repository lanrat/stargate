//go:build !linux && !freebsd

package stargate

import "syscall"

// controlFreebind is nil on unsupported platforms (non-Linux, non-FreeBSD).
// On these platforms, binding to non-local IP addresses is not supported.
var controlFreebind func(network, address string, c syscall.RawConn) error = nil
