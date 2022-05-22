package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/haxii/socks5"
	"github.com/lanrat/stargate/wireguard"
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
	wg       *wireguard.WG
)

const (
	maxProxies = 10000
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

	// if len(*localSubnet) != 0 && len(*wgConfigFile) != 0 {
	// 	fmt.Fprintf(os.Stderr, "Must pass subnet or wireguard config, not both\n")
	// 	flag.Usage()
	// 	return
	// }

	if len(*wgConfigFile) > 0 {
		wgConf, err := wireguard.ParseConfig(*wgConfigFile)
		check(err)
		*localSubnet = wgConf.Interface.AddrString[0] // TODO set the subnet for the rest of the code.. hax...
		log.Printf("WG Config: %+v", *wgConf)
		wg, err = wireguard.Start(*wgConf)
		check(err)
		err = wg.TestPing()
		check(err)
	}

	// log.Printf("waiting...")
	// time.Sleep(time.Second * 5)

	if *port == 0 && *random == 0 {
		l.Fatal("no SOCKS proxy ports provided, pass -port and/or -random")
	}

	_, cidr, err := net.ParseCIDR(*localSubnet)
	check(err)

	// calculate number of proxies about to start
	// show warning if too large
	subnetSize := maskSize(&cidr.Mask)
	v("subnet size %s", subnetSize.String())

	// prep network aware resolver
	resolver = &DNSResolver{
		network: getIPNetwork(&cidr.IP),
	}

	var work errgroup.Group
	// start proxies for port range
	if *port != 0 {
		// show warning if subnet too large
		if subnetSize.Cmp(big.NewInt(math.MaxInt32)) > 0 {
			l.Fatalf("proxy range provided larger than MaxInt32")
		}
		if subnetSize.Cmp(big.NewInt(maxProxies)) > 0 {
			l.Fatalf("proxy range provided too large %s > %d", subnetSize.String(), maxProxies)
		}

		ipList, err := hosts(cidr)
		check(err)

		// check that random port is outside range of other proxies
		if *random != 0 && *random >= *port && int(*random) < (int(*port)+len(ipList)) {
			l.Fatalf("random port %d inside range %d-%d", *random, *port, int(*port)+len(ipList))
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
	if *random != 0 && wg == nil {
		rand.Seed(time.Now().Unix())
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting random egress proxy %s\n", addrStr)
			return runRandomProxy(cidr, addrStr)
		})
	}

	// start wireguard proxy
	// TODO merge into random
	if *random != 0 && wg != nil {
		rand.Seed(time.Now().Unix())
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting wg egress proxy %s\n", addrStr)
			proxyIP := wg.Config.Interface.Address[0].As16()
			return runWgProxy(proxyIP[:], addrStr, wg.Net)
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
