package wg

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/netip"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

const (
	// ProtocolIPv4ICMP is IANA ICMP IPv4
	ProtocolIPv4ICMP = 1
	// ProtocolIPv6ICMP is IANA ICMP IPv6
	ProtocolIPv6ICMP = 58
)

type WG struct {
	Net    *netstack.Net
	Config *Config
}

func Start(cfg Config) (*WG, error) {
	iface := cfg.Interface

	//tun.CreateTUN("tun0",)

	tun, tnet, err := netstack.CreateNetTUN(iface.Address, iface.DNS, iface.MTU)
	if err != nil {
		return nil, err
	}
	logLevel := device.LogLevelError
	//logLevel = device.LogLevelVerbose
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(logLevel, "WG:"))
	ipcStr := cfg.getIPC()
	//log.Printf("DEBUG, ipcStr: \n%s", ipcStr)
	err = dev.IpcSet(ipcStr)
	if err != nil {
		return nil, err
	}

	err = dev.Up()
	if err != nil {
		return nil, err
	}

	return &WG{
		Net:    tnet,
		Config: &cfg,
	}, nil
}

func (w *WG) TestPing(ctx context.Context, host netip.Addr) error {
	protocol := ProtocolIPv4ICMP
	if host.Is6() {
		protocol = ProtocolIPv6ICMP
	}

	socket, err := w.Net.DialContext(ctx, "ping", host.String())
	if err != nil {
		return err
	}
	requestPing := icmp.Echo{
		Seq:  rand.Intn(1 << 16),
		Data: []byte("indeed"),
	}
	var icmpType icmp.Type = ipv4.ICMPTypeEcho
	if protocol == ProtocolIPv6ICMP {
		icmpType = ipv6.ICMPTypeEchoRequest
	}
	icmpBytes, err := (&icmp.Message{Type: icmpType, Code: 0, Body: &requestPing}).Marshal(nil)
	if err != nil {
		return err
	}
	socket.SetReadDeadline(time.Now().Add(time.Second * 10))
	//start := time.Now()
	_, err = socket.Write(icmpBytes)
	if err != nil {
		return err
	}
	n, err := socket.Read(icmpBytes[:])
	if err != nil {
		return err
	}
	replyPacket, err := icmp.ParseMessage(protocol, icmpBytes[:n])
	if err != nil {
		return err
	}
	replyPing, ok := replyPacket.Body.(*icmp.Echo)
	if !ok {
		return fmt.Errorf("invalid reply type: %v", replyPacket)
	}
	if !bytes.Equal(replyPing.Data, requestPing.Data) || replyPing.Seq != requestPing.Seq {
		return fmt.Errorf("invalid ping reply: %v", replyPing)
	}
	//log.Printf("DEBUG Ping latency: %v", time.Since(start))
	return nil
}
