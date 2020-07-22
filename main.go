package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/haxii/socks5"
	"golang.org/x/sync/errgroup"
)

var (
	l        *log.Logger
	resolver socks5.NameResolver

	listenIP = flag.String("listen", "127.0.0.1", "IP to listen on")
	port     = flag.Uint("port", 0, "first port to start listening on")
	proxy    = flag.String("proxy", "127.0.0.1/32", "CIDR notation of proxy IPs")
	random   = flag.Uint("random", 0, "port to use for random proxy server")
	verbose  = flag.Bool("verbose", false, "enable verbose logging")
)

const (
	maxProxies = 10000
)

func main() {
	flag.Parse()
	l = log.New(os.Stderr, "", log.LstdFlags)

	if *port == 0 && *random == 0 {
		l.Fatal("no SOCKS proxy ports provided, pass -port and/or -random")
	}

	_, cidr, err := net.ParseCIDR(*proxy)
	check(err)

	// calculate number of proxies about to start
	// show warning if too large
	subnetSize := maskSize(&cidr.Mask)
	if subnetSize < 0 {
		l.Fatalf("proxy range provided larger than max int64")
	}

	// prep network aware resolver
	resolver = &DNSResolver{
		network: getIPNetwork(&cidr.IP),
	}

	var work errgroup.Group
	if *port != 0 {
		// show warning if subnet too large
		if subnetSize > maxProxies {
			l.Fatalf("proxy range provided too large %d > %d", subnetSize, maxProxies)
		}

		ips, err := hosts(cidr)
		check(err)

		// check that random port is outside range of other proxies
		if *random != 0 && *random >= *port && int(*random) < (int(*port)+len(ips)) {
			l.Fatalf("random port %d inside range %d-%d", *random, *port, int(*port)+len(ips))
		}

		// convert all IPs to addressess
		addresses, err := ips2Address(ips)
		check(err)

		l.Printf("starting on %d IPs\n", len(addresses))
		for num, address := range addresses {
			listenPort := num + int(*port)
			address := address // https://golang.org/doc/faq#closures_and_goroutines
			work.Go(func() error {
				addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(listenPort))
				l.Printf("Starting proxy %s on IP: %s\n", addrStr, address.String())
				return runProxy(address, addrStr)
			})
		}
	}

	// start random proxy if -random set
	if *random != 0 {
		rand.Seed(time.Now().Unix())
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting random egress proxy %s\n", addrStr)
			return runRandomProxy(cidr, addrStr)
		})
	}

	err = work.Wait()
	check(err)
}

func check(err error) {
	if err != nil {
		l.Fatal(err)
	}
}

func v(format string, a ...interface{}) {
	if *verbose {
		log.Printf(format, a...)
	}
}
