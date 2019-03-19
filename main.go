package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/sync/errgroup"
)

var (
	l    *log.Logger
	work errgroup.Group

	listenIP = flag.String("listen", "127.0.0.1", "IP to listen on")
	port     = flag.Uint("port", 10000, "first port to start listening on")
	proxy    = flag.String("proxy", "127.0.0.1/32", "CIDR notation of proxy IPs")
)

func init() {
	flag.Parse()
	l = log.New(os.Stderr, "", log.LstdFlags)
}

func main() {
	ips, err := Hosts(*proxy)
	check(err)

	l.Printf("starting on %d IPs\n", len(ips))
	for num, ip := range ips {
		listenPort := num + int(*port)
		ip := ip // https://golang.org/doc/faq#closures_and_goroutines
		work.Go(func() error {
			l.Printf("Starting proxy %s:%d on IP: %s\n", *listenIP, listenPort, ip)
			return runProxy(ip, fmt.Sprintf("%s:%d", *listenIP, listenPort))
		})
	}

	err = work.Wait()
	check(err)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
