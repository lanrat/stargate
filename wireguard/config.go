package wireguard

import (
	"encoding/base64"
	"fmt"
	"net/netip"
	"strings"

	"gopkg.in/ini.v1"
)

const defaultKeepAlive = 25
const defaultMTU = 1420

type Config struct {
	Interface InterfaceConfig
	Peer      PeerConfig
}

func (c *Config) getIPC() string {
	request := fmt.Sprintf(`private_key=%x
public_key=%064x
endpoint=%s
persistent_keepalive_interval=%d
preshared_key=%064x
allowed_ip=0.0.0.0/0
allowed_ip=::0/0`,
		c.Interface.PrivateKey, c.Peer.PublicKey, c.Peer.Endpoint, c.Peer.PersistentKeepalive, c.Peer.PreSharedKey)
	// TODO set AllowedIP correctly (requires correct subnet parsing for config)
	return request
}

type InterfaceConfig struct {
	PrivateKey []byte
	Address    []netip.Addr
	DNS        []netip.Addr
	MTU        int
	AddrString []string
}

type InterfaceConfigIni struct {
	PrivateKey string
	Address    []string
	DNS        []string
	MTU        int
}

func (i *InterfaceConfigIni) toConfig() (InterfaceConfig, error) {
	var c InterfaceConfig
	var err error
	c.PrivateKey, err = base64.StdEncoding.DecodeString(i.PrivateKey)
	if err != nil {
		return c, err
	}
	c.Address, err = strToAddrs(i.Address)
	if err != nil {
		return c, err
	}
	c.AddrString = i.Address
	c.DNS, err = strToAddrs(i.DNS)
	if err != nil {
		return c, err
	}
	if i.MTU == 0 {
		i.MTU = defaultMTU
	}
	c.MTU = i.MTU
	return c, nil
}

type PeerConfig struct {
	PublicKey           []byte
	AllowedIPs          []netip.Addr
	PreSharedKey        []byte
	Endpoint            string
	PersistentKeepalive int
}

type PeerConfigIni struct {
	PublicKey           string
	AllowedIPs          []string
	PreSharedKey        string
	Endpoint            string
	PersistentKeepalive int
}

func (p *PeerConfigIni) toConfig() (PeerConfig, error) {
	var c PeerConfig
	var err error
	c.PublicKey, err = base64.StdEncoding.DecodeString(p.PublicKey)
	if err != nil {
		return c, err
	}
	c.AllowedIPs, err = strToAddrs(p.AllowedIPs)
	if err != nil {
		return c, err
	}
	c.PreSharedKey, err = base64.StdEncoding.DecodeString(p.PreSharedKey)
	if err != nil {
		return c, err
	}
	c.Endpoint = p.Endpoint
	if p.PersistentKeepalive == 0 {
		p.PersistentKeepalive = defaultKeepAlive
	}
	c.PersistentKeepalive = p.PersistentKeepalive
	return c, nil
}

func strToAddrs(s []string) ([]netip.Addr, error) {
	out := make([]netip.Addr, 0, 1)
	for _, part := range s {
		part = strings.TrimSpace(part)
		part = strings.SplitN(part, "/", 2)[0] // remove subnet if provided
		// TODO not sure how well this handles subnets...
		addr, err := netip.ParseAddr(part)
		if err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	return out, nil
}

// ParseConfig takes the path of a configuration file and parses it into Configuration
func ParseConfig(path string) (*Config, error) {
	iniOpt := ini.LoadOptions{
		Insensitive:  true,
		AllowShadows: true,
	}
	iniCfg, err := ini.LoadSources(iniOpt, path)
	if err != nil {
		return nil, err
	}
	cfg := new(Config)

	// Get Interfaces
	ifaceSection, err := iniCfg.GetSection("Interface")
	if err != nil {
		return nil, err
	}
	interfaceIni := new(InterfaceConfigIni)
	err = ifaceSection.MapTo(interfaceIni)
	if err != nil {
		return nil, err
	}
	cfg.Interface, err = interfaceIni.toConfig()
	if err != nil {
		return nil, err
	}

	peerSection, err := iniCfg.GetSection("Peer")
	if err != nil {
		return nil, err
	}
	peerIni := new(PeerConfigIni)
	err = peerSection.MapTo(peerIni)
	if err != nil {
		return nil, err
	}
	cfg.Peer, err = peerIni.toConfig()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
