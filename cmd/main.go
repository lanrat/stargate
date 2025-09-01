// Package main implements Stargate, a TCP SOCKS proxy server that can egress
// traffic from multiple IP addresses within a subnet.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"strings"

	"github.com/lanrat/stargate"
)

// Command-line flags
var (
	listenAddr   = flag.String("listen", "127.0.0.1:1080", "listen on specified IP:port (e.g., '127.0.0.1:1337', '127.0.0.1:8080', '[::1]:1080').")
	subnetBits   = flag.Uint("subnet-size", 0, "CIDR prefix length for random subnet proxy (e.g., 64 for /64 IPv6 subnets)")
	verbose      = flag.Bool("verbose", false, "enable verbose logging")
	printVersion = flag.Bool("version", false, "print version and exit")
	runTest      = flag.Bool("test", false, "run test request on all IPs and exit")
)

// Global variables
var (
	// l is the logger instance used throughout the application
	l = log.New(os.Stderr, "", log.LstdFlags)
	// version is the application version string, set at build time
	version = "dev"
)

// main is the entry point of the Stargate proxy server.
// It parses command-line flags, validates the CIDR configuration,
// and starts either the test mode or the SOCKS5 proxy server.
func main() {
	// Set custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [OPTION]... CIDR\n\tCIDR example: \"192.0.2.0/24\"\nOPTIONS:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	// check for version flag
	if *printVersion {
		fmt.Println(showVersion())
		return
	}

	// require CIDR argument
	if flag.NArg() != 1 {
		flag.Usage()
		return
	}
	cidrStr := flag.Arg(0)
	parsedNetwork, err := netip.ParsePrefix(cidrStr)
	if err != nil {
		l.Fatal(err)
	}

	if !parsedNetwork.IsValid() {
		l.Fatalf("parsed CIDR: %s is not valid", parsedNetwork.String())
	}

	var maxNetworkBits uint = 32
	if parsedNetwork.Addr().Is6() {
		maxNetworkBits = 128
	}

	// if unset, set to /32 for IPv4 or /128 for IPv6
	cidrBits := parsedNetwork.Bits()
	if *subnetBits == 0 {
		*subnetBits = maxNetworkBits
	} else {
		if *subnetBits < uint(cidrBits) {
			l.Fatalf("passed subnet-size %d must be >= CIDR netmask %d", *subnetBits, cidrBits)
		}
	}

	// for now, we only support up to 2^64 different subnet hosts
	// more could be supported by switching from uint64 to big.Int types.
	// for even with IPv6, using more than 2^64 networks is incredibly unlikely
	// if someone dares to do this, error
	totalNetworksBits := *subnetBits - uint(cidrBits)
	if totalNetworksBits > 64 {
		l.Fatalf("subnet host range too large. Can't run on over 2^64 networks, got 2^%d", totalNetworksBits)
	}

	// set stargate Logger
	stargate.Logger = v

	// check for IP conflicts
	conflicts, err := stargate.CheckHostConflicts(&parsedNetwork)
	if err != nil {
		l.Fatal(err)
	}
	for _, ip := range conflicts {
		l.Printf("Warning: possible IP conflict on %s", ip)
	}

	hostsPerNetwork := 1 << (maxNetworkBits - *subnetBits)
	totalNetworks := 1 << totalNetworksBits
	l.Printf("Running with subnet size /%d and /%d prefix resulting in %d egress networks and %d options per network", *subnetBits, cidrBits, totalNetworks, hostsPerNetwork)

	// test mode
	if *runTest {
		// test requests
		err := test(context.Background(), parsedNetwork, *subnetBits)
		if err != nil {
			l.Fatal(err)
		}
		l.Printf("All Tests Pass!")
		return
	}

	// Parse address to handle both "port" and "ip:port" formats
	if !strings.Contains(*listenAddr, ":") {
		// Just port, bind to all interfaces
		*listenAddr = ":" + *listenAddr
	}

	// run subnet proxy server
	l.Printf("Starting subnet egress proxy %s\n", *listenAddr)
	err = runRandomSubnetProxy(*listenAddr, parsedNetwork, *subnetBits)
	if err != nil {
		l.Fatal(err)
	}
}

// v logs a message if verbose logging is enabled.
func v(format string, a ...any) {
	if *verbose {
		l.Printf(format, a...)
	}
}

// showVersion returns a formatted version string for display.
func showVersion() string {
	return fmt.Sprintf("Version: %s", version)
}
