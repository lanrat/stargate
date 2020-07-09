package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	l *log.Logger

	listenIP = flag.String("listen", "127.0.0.1", "IP to listen on")
	port     = flag.Uint("port", 0, "first port to start listening on")
	proxy    = flag.String("proxy", "127.0.0.1/32", "CIDR notation of proxy IPs")
	random   = flag.Uint("random", 0, "port to use for random proxy server")
	udp      = flag.Bool("udp", false, "run in UDP mode with support for associate")
)

func main() {
	flag.Parse()
	l = log.New(os.Stderr, "", log.LstdFlags)

	if *port == 0 && *random == 0 {
		l.Fatal("no SOCKS proxy ports provided, pass -port and/or -random")
	}

	ips, err := hosts(*proxy)
	check(err)

	// check that random port is outside range of other proxies
	if *port != 0 && *random != 0 && *random >= *port && int(*random) < (int(*port)+len(ips)) {
		l.Fatalf("random port %d inside range %d-%d", *random, *port, int(*port)+len(ips))
	}

	// convert all IPs to addressess
	addresses, err := ips2Address(ips)
	check(err)

	var work errgroup.Group
	if *port != 0 {
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

	// start random proxy if set
	if *random != 0 {
		rand.Seed(time.Now().Unix())
		work.Go(func() error {
			addrStr := net.JoinHostPort(*listenIP, strconv.Itoa(int(*random)))
			l.Printf("Starting random egress proxy %s\n", addrStr)
			return runRandomProxy(addresses, addrStr)
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
