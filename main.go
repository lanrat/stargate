package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"strconv"
	"time"

	"github.com/haxii/socks5"
	"github.com/lanrat/stargate/wg"
	"golang.org/x/sync/errgroup"
)

// flags
var (
	listenIP     = flag.String("listen", "localhost", "IP to listen on")
	port         = flag.Uint("port", 0, "first port to start listening on")
	random       = flag.Uint("random", 0, "port to use for random proxy server")
	verbose      = flag.Bool("verbose", false, "enable verbose logging")
	wgConfigFile = flag.String("wireguard", "", "wireguard config file")
	localSubnet  = flag.String("subnet", "", "local subnet to proxy")
)

var (
	l        = log.New(os.Stderr, "", log.LstdFlags)
	resolver socks5.NameResolver
	wgNet    *wg.WG
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [OPTION]... CIDR\n\tCIDR example: -subnet \"192.0.2.0/24\"\nOPTIONS:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(*localSubnet) != 0 && len(*wgConfigFile) != 0 {
		fmt.Fprintf(os.Stderr, "Must pass subnet or wireguard config\n\n")
		flag.Usage()
		return
	}

	if len(*wgConfigFile) > 0 {
		wgConf, err := wg.ParseConfig(*wgConfigFile)
		check(err)
		*localSubnet = wgConf.Interface.AddrString[0] // TODO set the subnet for the rest of the code.. hax...
		//log.Printf("WG Config: %+v", *wgConf)
		wgNet, err = wg.Start(*wgConf)
		check(err)
		err = wgNet.Net.Spoof(1)
		check(err)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		err = wgNet.TestPing(ctx, netip.MustParseAddr("2001:4860:4860::8888"))
		cancel()
		check(err)
	}

	if *port == 0 && *random == 0 {
		fmt.Fprintf(os.Stderr, "no SOCKS proxy ports provided, pass -port and/or -random\n")
		flag.Usage()
		return
	}

	_, cidr, err := net.ParseCIDR(*localSubnet)
	check(err)

	// TODO test that the cidr is allowed to be used on this machine

	// calculate number of proxies about to start
	// show warning if too large
	subnetSize := maskSize(&cidr.Mask)
	v("subnet size %s", subnetSize.String())

	// prep network aware resolver
	resolver = &DNSResolver{
		network: getIPNetwork(&cidr.IP),
	}

	// prep random
	rand.Seed(time.Now().Unix())

	var work errgroup.Group
	// start proxies for port range
	if *port != 0 {

		ipList, err := hosts(cidr)
		check(err)

		highPort := int(*port) + len(ipList) - 1
		// check if subnet too large
		if highPort > math.MaxUint16 {
			l.Fatalf("last proxy port %d is higher than highest allowed port (MaxUint16)", highPort)
		}

		// check that random port is outside range of other proxies
		if (*random != 0) && (*random >= *port) && (int(*random) <= highPort) {
			l.Fatalf("random port %d inside range %d-%d", *random, *port, highPort)
		}

		l.Printf("starting on %s\n", cidr.String())
		started := 0
		for num, ip := range ipList {
			listenPort := num + int(*port)
			ip := ip // https://golang.org/doc/faq#closures_and_goroutines
			started++

			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(listenPort))
			l.Printf("Starting proxy %s using IP: %s\n", addrStr, ip.String())
			work.Go(func() error {
				return runProxy(ip, addrStr)
			})
		}
		l.Printf("started %d proxies\n", started)
	}

	// start random proxy if -random set
	if *random != 0 && wgNet == nil {
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting random egress proxy %s\n", addrStr)
			return runRandomProxy(cidr, addrStr)
		})
	}

	// start wireguard proxy
	// TODO merge into random
	if *random != 0 && wgNet != nil {
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting wg egress proxy %s\n", addrStr)
			//proxyIP := wg.Config.Interface.Address[1].As16()
			return runWgProxy(cidr, addrStr, wgNet.Net)
		})
	}

	err = work.Wait()
	check(err)
}

// check checks errors
func check(err error) {
	if err != nil {
		l.Fatal(err)
	}
}

// v verbose logging
func v(format string, a ...interface{}) {
	if *verbose {
		l.Printf(format, a...)
	}
}
