# Stargate

[![Go Report Card](https://goreportcard.com/badge/github.com/lanrat/stargate)](https://goreportcard.com/report/github.com/lanrat/stargate)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/lanrat/stargate)](https://pkg.go.dev/github.com/lanrat/stargate)
[![Docker](https://github.com/lanrat/stargate/actions/workflows/docker.yml/badge.svg)](https://github.com/lanrat/stargate/actions/workflows/docker.yml)

Stargate is both a Go library and a TCP SOCKS5 proxy server that can egress traffic from multiple IP addresses within a subnet. It randomly distributes connections across different IP addresses to help avoid rate-limiting and provide load balancing across your available IP range.

This requires the host running stargate to have the subnet routed directly to it.

If you have an IPv6 subnet, stargate can allow you to make full use of it by any program that can speak SOCKS, or you can use the library directly in your Go applications.

## Usage

```console
Usage of ./stargate: [OPTION]... CIDR
 CIDR example: "192.0.2.0/24"
OPTIONS:
  -listen string
     listen on specified [IP:]port (e.g., '1337', '127.0.0.1:8080', '[::1]:1080') (default "localhost:1080")
  -subnet-size uint
     CIDR prefix length for random subnet proxy (e.g., 64 for /64 IPv6 subnets)
  -test
     run test request on all IPs and exit
  -verbose
     enable verbose logging
  -version
     print version and exit
```

Stargate operates as a single SOCKS5 proxy server that randomly selects egress IP addresses from your specified CIDR range. This approach is much more memory-efficient and suitable for large IPv6 ranges.

## Test Flag - Preventing IP Address Leakage

**IMPORTANT**: Before using Stargate in production, always run the test mode first to ensure there are no unintended IP address leaks.

The `-test` flag performs comprehensive validation by:

- Testing HTTP requests from every available IP address in your CIDR range
- Verifying that each egress IP matches the intended source address
- Detecting binding errors or network misconfigurations
- Ensuring no connections leak through unintended IP addresses

**Note:** When using `-subnet-size`, the test will validate one randomly selected IP address from each subnet rather than testing every possible IP. For example, with `-subnet-size 64` on a /48, it tests one IP per /64 subnet, not every IP in the entire /48. In order to test every possible IP address, do not pass a `-subnet-size` option when using `-test`.

**Always run this test before production use:**

```bash
# Test your configuration first - THIS IS CRITICAL!
./stargate -test 192.0.2.0/24

# Only proceed to normal operation after tests pass
./stargate 192.0.2.0/24
```

The test will fail immediately if any IP address binding issues are detected, preventing potential IP leakage that could compromise your setup.

## Subnet Distribution

When used with `-subnet-size`, the proxy will randomly distribute connections across different subnets within the main CIDR range. For example, with a /48 IPv6 block and `-subnet-size 64`, connections will be distributed across random /64 subnets, giving you access to multiple /64 networks within your larger allocation.

## Examples

### Basic Usage

Start a SOCKS5 proxy on the default port (1080) that randomly egresses from IPs in the 192.0.2.0/24 range:

```bash
# Always test first!
./stargate -test 192.0.2.0/24

# Run the proxy after tests pass
./stargate 192.0.2.0/24
```

### Custom Listen Address

Start a SOCKS5 proxy listening on a specific IP and port:

```bash
./stargate -listen 127.0.0.7:8080 192.0.2.0/24
```

### IPv6 with Large Address Space

Use an IPv6 /64 subnet - this gives you 2^64 possible egress IPs:

```bash
./stargate -test 2001:DB8:1337::/64
./stargate 2001:DB8:1337::/64
```

### Subnet-Level Distribution

Distribute connections across multiple /64 subnets within a /48 IPv6 allocation:

```bash
./stargate -test -subnet-size 64 2001:DB8:1337::/48
./stargate -subnet-size 64 2001:DB8:1337::/48
```

This will randomly select from different /64 networks within your /48, providing both IP and subnet-level distribution.

## Routing Setup

Before Stargate can use IP addresses from your subnet, you need to configure your network to route that subnet to the host running Stargate.

**Step 1: Configure subnet routing**
First, ensure your network infrastructure routes the entire subnet to your host machine. This is typically done at your router or network provider level.

**Step 2: Add local route**
Once the subnet is routed to your host, tell your operating system that it can bind to any IP in that range by adding a local route:

```shell
# Example: if 192.0.2.0/24 is routed to interface eth0
ip -4 route add local 192.0.2.0/24 dev eth0
```

**Important:** Don't assign individual IPs from the subnet directly to your network interface. This prevents the kernel from reserving addresses as broadcast addresses and keeps them available for Stargate.

**Step 3: Enable non-local binding (if needed)**
Some systems require enabling non-local binding to allow applications to bind to IPs that aren't directly assigned to interfaces:

```shell
# For IPv4
sysctl net.ipv4.ip_nonlocal_bind=1
# For IPv6
sysctl net.ipv6.ip_nonlocal_bind=1
```

## Download

### [Precompiled Binaries](https://github.com/lanrat/stargate/releases)

The easiest way to get started is with [precompiled binaries](https://github.com/lanrat/stargate/releases) available for multiple platforms including Linux and FreeBSD. These are statically linked and ready to run without additional dependencies.

### [Docker](https://github.com/lanrat/stargate/pkgs/container/stargate)

Docker is particularly useful for deployment in containerized environments, though network configuration requires special attention to ensure proper subnet routing.

Running in docker will require `--net=host`, or the subnet must be routed directly to the container.

```shell
docker pull ghcr.io/lanrat/stargate:latest
```

## Building

Building from source is straightforward - just run `make`!

```bash
git clone https://github.com/lanrat/stargate.git
cd stargate
make
```

This will produce a statically linked binary that's ready to use.

### Platforms

Stargate makes use of freebind, which allowed for creating a socket with a source IP address on an interface that does not have that IP directly bound to it. This is only available on Linux and FreeBSD. Stargate can work on other platforms, but it will require every address to be previously bound to the interface before running.

## Go Library

You can use Stargate as a Go library in your applications to make HTTP requests or other network connections using random source IP addresses:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "net/netip"
    "time"

    "github.com/lanrat/stargate"
)

func main() {
    // Parse your CIDR range
    prefix, err := netip.ParsePrefix("192.0.2.0/24")
    if err != nil {
        panic(err)
    }

    // Create a RandomIPDialer for the subnet
    dialer, err := stargate.NewRandomIPIterator(prefix, 32)
    if err != nil {
        panic(err)
    }

    // Create an HTTP client that uses random source IPs
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: dialer.Dial,
        },
        Timeout: 10 * time.Second,
    }

    // Make requests - each will use a different random IP
    for i := 0; i < 5; i++ {
        resp, err := client.Get("https://httpbin.org/ip")
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i+1, err)
            continue
        }
        resp.Body.Close()
        fmt.Printf("Request %d: Status %d\n", i+1, resp.StatusCode)
    }
}
```

You can also get the next random IP and DialFunc separately:

```go
// Get next random IP and dialer function
ip, dialFunc, err := dialer.NextDial()
if err != nil {
    panic(err)
}

fmt.Printf("Next egress IP will be: %s\n", ip)

// Use the dial function directly
conn, err := dialFunc(context.Background(), "tcp", "example.com:80")
if err != nil {
    panic(err)
}
defer conn.Close()
```
