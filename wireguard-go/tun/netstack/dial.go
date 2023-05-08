package netstack

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type tcpError struct {
	err tcpip.Error
}

func (t *tcpError) Error() string {
	if t.err == nil {
		return ""
	}
	return t.err.String()
}

func (n *Net) Spoof(i tcpip.NICID) error {
	err := n.stack.SetSpoofing(i, true)
	if err != nil {
		tErr := tcpError{err: err}
		return &tErr
	}
	return nil
}

func (n *Net) DialTCPWithBindAddr(ctx context.Context, local, remote netip.AddrPort) (*gonet.TCPConn, error) {
	localAddr, _ := convertToFullAddr(local)
	remoteAddr, pn := convertToFullAddr(remote)
	return n.DialTCPWithBind(ctx, localAddr, remoteAddr, pn)
}

// DialTCPWithBind creates a new TCPConn connected to the specified
// remoteAddress with its local address bound to localAddr.
func (n *Net) DialTCPWithBind(ctx context.Context, localAddr, remoteAddr tcpip.FullAddress, network tcpip.NetworkProtocolNumber) (*gonet.TCPConn, error) {
	// Create TCP endpoint, then connect.
	var wq waiter.Queue
	ep, err := n.stack.NewEndpoint(tcp.ProtocolNumber, network, &wq)
	if err != nil {
		return nil, errors.New(err.String())
	}

	// Create wait queue entry that notifies a channel.
	//
	// We do this unconditionally as Connect will always return an error.
	waitEntry, notifyCh := waiter.NewChannelEntry(nil)
	wq.EventRegister(&waitEntry, waiter.WritableEvents)
	defer wq.EventUnregister(&waitEntry)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Bind before connect if requested.
	if localAddr != (tcpip.FullAddress{}) {
		if err = ep.Bind(localAddr); err != nil {
			return nil, fmt.Errorf("ep.Bind(%+v) = %s", localAddr, err)
		}
	}

	err = ep.Connect(remoteAddr)
	if _, ok := err.(*tcpip.ErrConnectStarted); ok {
		select {
		case <-ctx.Done():
			ep.Close()
			return nil, ctx.Err()
		case <-notifyCh:
		}

		err = ep.LastError()
	}
	if err != nil {
		ep.Close()
		return nil, &net.OpError{
			Op:   "connect",
			Net:  "tcp",
			Addr: fullToTCPAddr(remoteAddr),
			Err:  errors.New(err.String()),
		}
	}

	return gonet.NewTCPConn(&wq, ep), nil
}

func fullToTCPAddr(addr tcpip.FullAddress) *net.TCPAddr {
	return &net.TCPAddr{IP: net.IP(addr.Addr), Port: int(addr.Port)}
}
