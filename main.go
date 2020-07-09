package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	l *log.Logger

	listenIP = flag.String("listen", "127.0.0.1", "IP to listen on")
	port     = flag.Uint("port", 10000, "first port to start listening on")
	proxy    = flag.String("proxy", "127.0.0.1/32", "CIDR notation of proxy IPs")
	random   = flag.Uint("random", 0, "port to use for random proxy server")
)

func main() {
	flag.Parse()
	l = log.New(os.Stderr, "", log.LstdFlags)

	ips, err := hosts(*proxy)
	check(err)

	// check that random port is outside range of other proxies
	if *random != 0 && *random >= *port && int(*random) < (int(*port)+len(ips)) {
		l.Fatalf("random port %d inside range %d-%d", *random, *port, int(*port)+len(ips))
	}

	// convert all IPs to addressess
	addresses, err := ips2Address(ips)
	check(err)

	var work errgroup.Group
	l.Printf("starting on %d IPs\n", len(addresses))
	for num, address := range addresses {
		listenPort := num + int(*port)
		address := address // https://golang.org/doc/faq#closures_and_goroutines
		work.Go(func() error {
			l.Printf("Starting proxy %s:%d on IP: %s\n", *listenIP, listenPort, address.IP.String())
			return runProxy(address, fmt.Sprintf("%s:%d", *listenIP, listenPort))
		})
	}

	// start random proxy if set
	if *random != 0 {
		rand.Seed(time.Now().Unix())
		work.Go(func() error {
			l.Printf("Starting random egress proxy %s:%d\n", *listenIP, *random)
			return runRandomProxy(addresses, fmt.Sprintf("%s:%d", *listenIP, *random))
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
