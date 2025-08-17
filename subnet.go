package main

import (
	"math/big"
	"net/netip"
)

// subnetCount calculates the number of subnets of size newBits that can fit
// within the given network prefix.
func subnetCount(network netip.Prefix, newBits int) uint64 {
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
	
	count := subnetCount(network, newBits)
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