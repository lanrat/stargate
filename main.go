package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"net"
	"os"
	"strconv"

	"github.com/haxii/socks5"
	"golang.org/x/sync/errgroup"
)

// flags
var (
	listenIP     = flag.String("listen", "localhost", "IP to listen on")
	port         = flag.Uint("port", 0, "first port to start listening on")
	random       = flag.Uint("random", 0, "port to use for random proxy server")
	randsubnet   = flag.Uint("randsubnet", 0, "")
	verbose      = flag.Bool("verbose", false, "enable verbose logging")
	printVersion = flag.Bool("verbose", false, "enable verbose logging")
)

var (
	l        = log.New(os.Stderr, "", log.LstdFlags)
	resolver socks5.NameResolver
	version  = "dev"
)

const (
	maxProxies = 10000
)

func main() {
	flag.Parse()
	// check for version flag
	if *printVersion {
		fmt.Println(showVersion())
		return
	}
	if flag.NArg() != 1 {
		flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage of %s: [OPTION]... CIDR\n\tCIDR example: \"192.0.2.0/24\"\nOPTIONS:\n", os.Args[0])
			flag.PrintDefaults()
		}
		flag.Usage()
		return
	}
	proxy := flag.Arg(0)

	if *port == 0 && *random == 0 {
		l.Fatal("no SOCKS proxy ports provided, pass -port and/or -random")
	}

	_, cidr, err := net.ParseCIDR(proxy)
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

	// start random subnet proxy if -randsubnet set
	if *randsubnet != 0 && *random != 0 {
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting random subnet egress proxy %s\n", addrStr)
			return runRandomSubnetProxy(addrStr, proxy, *randsubnet)
		})
	}

	// start random proxy if -random set
	if *random != 0 && *randsubnet == 0 {
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting random egress proxy %s\n", addrStr)
			return runRandomProxy(cidr, addrStr)
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

// showVersion returns a formatted version string for display.
func showVersion() string {
	return fmt.Sprintf("Version: %s", version)
}
