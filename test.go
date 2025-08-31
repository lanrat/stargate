package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// testTimeout is the maximum time allowed for each individual test request
	testTimeout = 30 * time.Second
	// testParallel is the number of concurrent test workers to run
	testParallel = 10
)

// testDial represents a test configuration pairing an IP address with its corresponding dialer function.
type testDial struct {
	ip   net.IP
	dial DialFunc
}

// test validates that all IP addresses in the given CIDR range can successfully
// make HTTP requests and receive the expected source IP in the response.
// It uses cloudflare.com/cdn-cgi/trace to verify the egress IP address.
func test(ctx context.Context, parsedNetwork netip.Prefix, cidrSize uint) error {
	// Create iterator for all host indices
	ipItr, err := NewRandomIPIterator(parsedNetwork, cidrSize)
	if err != nil {
		return err
	}

	var failed atomic.Uint64
	var tested atomic.Uint64

	group, grpCtx := errgroup.WithContext(ctx)
	inputChan := make(chan *testDial, testParallel)

	// start input
	group.Go(func() error {
		defer close(inputChan)
		total := ipItr.Size()
		for i := uint64(0); i < total; i++ {
			select {
			case <-grpCtx.Done():
				return grpCtx.Err()
			default:
				ip, dial, err := ipItr.NextDial()
				if err != nil {
					return err
				}
				inputChan <- &testDial{
					ip:   ip,
					dial: dial,
				}
			}
		}
		return nil
	})

	// print testing status
	statusStop := make(chan bool)
	if !*verbose {
		defer func() { statusStop <- true }()
		go func() {
			for {
				select {
				case <-statusStop:
					fmt.Printf("\n") // Clear the status
					return
				default:
					testedCount := tested.Load()
					totalHosts := ipItr.Size()
					progress := float64(testedCount) / float64(totalHosts) * 100
					fmt.Printf("\r Testing %d/%d (%.1f%%) failures: %d", testedCount, totalHosts, progress, failed.Load())
				}
			}
		}()
	}

	// start workers
	for i := 0; i < testParallel; i++ {
		group.Go(func() error {
			for {
				select {
				case <-grpCtx.Done():
					return grpCtx.Err()
				case testDial, ok := <-inputChan:
					if !ok {
						// done
						return nil
					}
					v("testing source IP: %s", testDial.ip.String())
					tested.Add(1)
					if err := testWithDialer(ctx, testDial.dial, testDial.ip); err != nil {
						if !*verbose {
							fmt.Printf("\n") // Clear the status line before printing error
						}
						l.Printf("test failed for IP %s: %v", testDial.ip.String(), err)
						failed.Add(1)
						continue
					}
				}
			}
		})
	}

	// Wait for all goroutines to complete
	if err := group.Wait(); err != nil {
		return err
	}

	if failedCount := failed.Load(); failedCount > 0 {
		return fmt.Errorf("test finished with %d/%d failures", failedCount, ipItr.Size())
	}

	return nil
}

// testWithDialer performs an HTTP request using the provided dialer and verifies
// that the egress IP matches the expected IP address by querying cloudflare.com/cdn-cgi/trace.
func testWithDialer(ctx context.Context, dial DialFunc, expectedIP net.IP) error {
	ctx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://cloudflare.com/cdn-cgi/trace", nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: dial,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		// If bindError, then unwrap
		var bindErr *IPBindError
		if errors.As(err, &bindErr) {
			return bindErr
		}
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	lines := strings.Split(string(body), "\n")
	var ipStr string
	for _, line := range lines {
		if strings.HasPrefix(line, "ip=") {
			ipStr = strings.TrimPrefix(line, "ip=")
			break
		}
	}
	if ipStr == "" {
		return fmt.Errorf("IP field not found in response")
	}
	ip := net.ParseIP(ipStr)
	if !expectedIP.Equal(ip) {
		return fmt.Errorf("test returned unexpected IP, expected %s, got %s", expectedIP, ip)
	}
	return nil
}
