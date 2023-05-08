module github.com/lanrat/stargate

require (
	github.com/haxii/socks5 v1.0.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/sync v0.0.0-20220513210516-0976fa681c29
	golang.zx2c4.com/wireguard v0.0.0-20220317000134-95b48cdb3961
	golang.zx2c4.com/wireguard/tun/netstack v0.0.0-20220407013110-ef5c587f782d
	gopkg.in/ini.v1 v1.66.4
	gvisor.dev/gvisor v0.0.0-20220520211104-bb1a83085b3b
// gvisor.dev/gvisor v0.0.0-20220520211629-7e72240f4f2e
)

//replace golang.zx2c4.com/wireguard => ./wireguard-go
//replace golang.zx2c4.com/wireguard/tun/netstack => ./wireguard-go/tun/netstack

go 1.13
