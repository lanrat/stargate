// Package stargate provides network connection functionality with randomized source IP addresses.
// It includes dialers that can bind to random IP addresses within specified CIDR ranges,
// with built-in verification to prevent IP address leaks and binding errors.
package stargate

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"net/netip"

	"github.com/lanrat/stargate/permute"
)

// IPBindError represents a critical error where a connection was bound to an unexpected IP address.
type IPBindError struct {
	IP    net.IP
	child error
}

// Error returns a formatted error message for the IP binding error.
func (e *IPBindError) Error() string {
	if e.child != nil {
		return e.child.Error()
	}
	return fmt.Sprintf("CRITICAL: Connection bind error on %s", e.IP)
}

// IPBindLeakError represents a critical error where a connection was bound to an unintended IP address.
// This error is used to prevent IP address leaks by aborting connections that don't use the expected source IP.
type IPBindLeakError struct {
	IPBindError
	ActualIP net.IP
}

// Error returns a formatted error message for the IP leak error.
func (e *IPBindLeakError) Error() string {
	return fmt.Sprintf("CRITICAL: Connection bound to %s instead of intended %s - aborting to prevent IP leak", e.ActualIP, e.IP)
}

// Unwrap returns the embedded IPBindError to support error unwrapping with errors.As.
func (e *IPBindLeakError) Unwrap() error {
	e.child = e
	return &e.IPBindError
}

// IPBindBroadcastError represents a critical error where a connection was bound to a broadcast IP address.
type IPBindBroadcastError struct {
	IPBindError
}

// Error returns a formatted error message for the broadcast IP binding error.
func (e *IPBindBroadcastError) Error() string {
	return fmt.Sprintf("CRITICAL: cant bind to broadcast address: %s", e.IP)
}

// Unwrap returns the embedded IPBindError to support error unwrapping with errors.As.
func (e *IPBindBroadcastError) Unwrap() error {
	e.child = e
	return &e.IPBindError
}

// DialFunc represents a function that establishes network connections.
// It follows the same signature as net.Dialer.DialContext.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// createDialerWithSourceIP creates a dialer that uses the specified IP as the source address
// and includes verification to ensure the connection is bound to the intended IP.
// It returns an error if the IP is a broadcast address or if the connection binds to an unexpected IP.
func createDialerWithSourceIP(ctx context.Context, network, addr string, sourceIP net.IP) (net.Conn, error) {
	v("dial %s from: %s to: %s", network, sourceIP.String(), addr)
	// check that we are not using a broadcast address
	if broadcastAddrs[sourceIP.String()] {
		return nil, &IPBindBroadcastError{
			IPBindError: IPBindError{IP: sourceIP},
		}
	}

	var localAddr net.Addr
	switch network {
	case "tcp":
		localAddr = &net.TCPAddr{
			IP: sourceIP,
		}
	case "udp":
		localAddr = &net.UDPAddr{
			IP: sourceIP,
		}
	default:
		return nil, fmt.Errorf("unknown network type %s", network)
	}

	// create the custom dialer
	d := net.Dialer{
		LocalAddr: localAddr,
		Control:   controlFreebind,
	}
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	// FAIL-SAFE: Verify the connection is using the intended IP
	var actualIP net.IP
	switch network {
	case "tcp":
		actualIP = conn.LocalAddr().(*net.TCPAddr).IP
	case "udp":
		actualIP = conn.LocalAddr().(*net.UDPAddr).IP
	default:
		return nil, fmt.Errorf("unknown network type %s: %s", network, conn.LocalAddr())
	}
	if !actualIP.Equal(sourceIP) {
		conn.Close()
		return nil, &IPBindLeakError{
			IPBindError: IPBindError{IP: sourceIP},
			ActualIP:    actualIP,
		}
	}
	v("verified connection bound to intended IP: %s", actualIP)
	return conn, nil
}

// randomIP generates a random IP address within the given CIDR range.
// It preserves the network portion and randomizes the host portion.
// Note: This may generate network or broadcast addresses, which are filtered
// by isValidHostIP() before use.
func randomIP(cidr *net.IPNet) net.IP {
	ip := make(net.IP, len(cidr.IP))
	copy(ip, cidr.IP)
	for i := range ip {
		rb := byte(rand.Intn(256))
		ip[i] = (cidr.Mask[i] & cidr.IP[i]) | (^cidr.Mask[i] & rb)
	}
	return ip
}

// RandomIPDialer manages iteration through random subnets within a CIDR range.
// It uses a permutation iterator to cycle through all possible subnets in a random order.
type RandomIPDialer struct {
	iterator    *permute.RandomParallelIterator
	prefix      netip.Prefix
	cidrBits    uint
	subnetCount uint64
}

// NewRandomIPIterator creates a new RandomIPDialer for the given network prefix.
// It calculates the number of possible subnets and initializes the random iterator.
func NewRandomIPIterator(prefix netip.Prefix, cidrBits uint) (*RandomIPDialer, error) {
	subnetCount := subnetCount64(prefix, int(cidrBits))
	if subnetCount == 0 {
		return nil, fmt.Errorf("subnet size is 0: %+v / %d", prefix, cidrBits)
	}
	v("creating NewRandomIPIterator network %s with a CIDR of %d, subnet pool size is %d", prefix, cidrBits, subnetCount)
	it := &RandomIPDialer{
		prefix:      prefix,
		cidrBits:    cidrBits,
		subnetCount: subnetCount,
	}

	// Create a RandomParallelIterator for subnet indices
	err := it.reset()
	if err != nil {
		return nil, err
	}
	return it, nil
}

// reset reinitializes the internal iterator to start over when all subnets have been used.
func (it *RandomIPDialer) reset() error {
	var err error
	it.iterator, err = permute.NewRandomParallelIterator(big.NewInt(0), new(big.Int).SetUint64(it.subnetCount))
	return err
}

// NextIP returns the next random subnet as a net.IPNet.
// When all subnets have been used, it automatically resets to start over.
func (it *RandomIPDialer) NextIP() (*net.IPNet, error) {
	var err error
	// Get next random subnet index
	index, ok := it.iterator.Next()
	if !ok {
		// All subnets have been used, create a new iterator to start over
		v("used all the subnets in our pool, looping back around...")
		err = it.reset()
		if err != nil {
			return nil, err
		}
		index, _ = it.iterator.Next()
	}

	// Get the subnet at this index
	subnetPrefix, ok := nthSubnet(it.prefix, int(it.cidrBits), index.Uint64())
	if !ok {
		return nil, fmt.Errorf("failed to get subnet at index %s", index.String())
	}

	// Convert netip.Prefix to net.IPNet for use with randomIP
	ipAddr := subnetPrefix.Addr()
	var subnet *net.IPNet
	if ipAddr.Is4() {
		ipv4 := ipAddr.As4()
		subnet = &net.IPNet{
			IP:   net.IP(ipv4[:]),
			Mask: net.CIDRMask(subnetPrefix.Bits(), 32),
		}
	} else {
		ipv6 := ipAddr.As16()
		subnet = &net.IPNet{
			IP:   net.IP(ipv6[:]),
			Mask: net.CIDRMask(subnetPrefix.Bits(), 128),
		}
	}

	return subnet, nil
}

// Size returns the total number of subnets available for iteration.
func (it *RandomIPDialer) Size() uint64 {
	return it.subnetCount
}

// Dial implements DialFunc and establishes a connection using a random egress IP.
func (it *RandomIPDialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	_, dial, err := it.NextDial()
	if err != nil {
		return nil, err
	}
	return dial(ctx, network, addr)
}

// NextDial returns the next random IP and a corresponding DialFunc for establishing connections.
func (it *RandomIPDialer) NextDial() (net.IP, DialFunc, error) {
	subnet, err := it.NextIP()
	if err != nil {
		return nil, nil, err
	}

	ip := randomIP(subnet)
	d := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return createDialerWithSourceIP(ctx, network, addr, ip)
	}
	return ip, d, nil
}

// subnetCount calculates the number of subnets of size newBits that can fit
// within the given network prefix.
// In the event > 2^64 networks is wanted, this needs to be updated to return a big.Int
func subnetCount64(network netip.Prefix, newBits int) uint64 {
	originalBits := network.Bits()
	if newBits <= originalBits {
		return 0
	}

	additionalBits := newBits - originalBits
	if network.Addr().Is4() {
		if additionalBits > 32 {
			return 0
		}
	} else {
		if additionalBits > 128 {
			return 0
		}
	}

	return 1 << additionalBits
}

// nthSubnet returns the nth subnet of size newBits within the given network prefix.
// Returns false if the subnet index is out of bounds or if newBits is smaller than
// the network's prefix length.
func nthSubnet(network netip.Prefix, newBits int, n uint64) (netip.Prefix, bool) {
	if newBits < network.Bits() {
		return netip.Prefix{}, false
	}

	count := subnetCount64(network, newBits)
	if count == 0 || n >= count {
		return netip.Prefix{}, false
	}

	baseAddr := network.Addr()
	if baseAddr.Is4() {
		return nthSubnetIPv4(network, newBits, n)
	}
	return nthSubnetIPv6(network, newBits, n)
}

// nthSubnetIPv4 calculates the nth IPv4 subnet of the specified size within the network.
// This is a helper function for nthSubnet that handles IPv4-specific logic.
func nthSubnetIPv4(network netip.Prefix, newBits int, n uint64) (netip.Prefix, bool) {
	baseAddr := network.Addr()
	if !baseAddr.Is4() {
		return netip.Prefix{}, false
	}

	as4 := baseAddr.As4()
	baseInt := uint32(as4[0])<<24 | uint32(as4[1])<<16 | uint32(as4[2])<<8 | uint32(as4[3])

	shift := 32 - newBits
	subnetInt := baseInt + (uint32(n) << shift)

	newAddr := netip.AddrFrom4([4]byte{
		byte(subnetInt >> 24),
		byte(subnetInt >> 16),
		byte(subnetInt >> 8),
		byte(subnetInt),
	})

	return netip.PrefixFrom(newAddr, newBits), true
}

// nthSubnetIPv6 calculates the nth IPv6 subnet of the specified size within the network.
// This is a helper function for nthSubnet that handles IPv6-specific logic using big.Int
// for the large address space calculations.
func nthSubnetIPv6(network netip.Prefix, newBits int, n uint64) (netip.Prefix, bool) {
	baseAddr := network.Addr()
	if !baseAddr.Is6() {
		return netip.Prefix{}, false
	}

	shift := 128 - newBits

	as16 := baseAddr.As16()
	baseInt := new(big.Int).SetBytes(as16[:])

	nBig := new(big.Int).SetUint64(n)
	nBig.Lsh(nBig, uint(shift))

	subnetInt := new(big.Int).Add(baseInt, nBig)

	bytes := subnetInt.Bytes()
	if len(bytes) > 16 {
		return netip.Prefix{}, false
	}

	var addr16 [16]byte
	copy(addr16[16-len(bytes):], bytes)

	newAddr := netip.AddrFrom16(addr16)
	return netip.PrefixFrom(newAddr, newBits), true
}
