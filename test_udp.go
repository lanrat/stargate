package main

import (
	"fmt"
	"log"
	"time"

	"github.com/miekg/dns"
	"github.com/txthinking/socks5"
)

func main() {
	proxyAddr := "127.0.0.1:1080"
	// dnsServer := "8.8.8.8:53"
	// queryDomain := "google.com"

	// https://www.dns.toys/
	//dig -4 ip @dns.toys
	dnsServer := "204.48.19.68:53"
	queryDomain := "ip"

	fmt.Printf("Testing SOCKS5 UDP support through proxy %s\n", proxyAddr)
	fmt.Printf("DNS Server: %s\n", dnsServer)
	fmt.Printf("Query Domain: %s\n", queryDomain)

	// Test TCP CONNECT first
	// fmt.Println("\n=== Testing TCP CONNECT ===")
	// testTCP(proxyAddr, dnsServer, queryDomain)

	// Test UDP ASSOCIATE
	fmt.Println("\n=== Testing UDP ASSOCIATE ===")
	testUDP(proxyAddr, dnsServer, queryDomain)

	// Test different DNS servers
	// fmt.Println("\n=== Testing Different DNS Servers ===")
	// testMultipleDNS(proxyAddr, queryDomain)
}

func testTCP(proxyAddr, dnsServer, queryDomain string) {
	// Create SOCKS5 client
	client, err := socks5.NewClient(proxyAddr, "", "", 0, 60)
	if err != nil {
		log.Printf("Failed to create SOCKS5 client: %v", err)
		return
	}

	// Test TCP connection to DNS server
	conn, err := client.Dial("tcp", dnsServer)
	if err != nil {
		log.Printf("Failed to dial TCP through SOCKS5: %v", err)
		return
	}
	defer conn.Close()

	// Create DNS query using miekg/dns
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(queryDomain), dns.TypeA)
	m.Id = 12345 // Set a recognizable ID

	// Pack the DNS message
	data, err := m.Pack()
	if err != nil {
		log.Printf("Failed to pack DNS query: %v", err)
		return
	}

	// TCP DNS requires 2-byte length prefix
	lengthPrefix := []byte{byte(len(data) >> 8), byte(len(data))}
	tcpQuery := append(lengthPrefix, data...)

	_, err = conn.Write(tcpQuery)
	if err != nil {
		log.Printf("Failed to send TCP DNS query: %v", err)
		return
	}

	// Read response length first
	lengthBuf := make([]byte, 2)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(lengthBuf)
	if err != nil {
		log.Printf("Failed to read TCP DNS response length: %v", err)
		return
	}

	// Read the actual response
	respLength := int(lengthBuf[0])<<8 | int(lengthBuf[1])
	response := make([]byte, respLength)
	n, err := conn.Read(response)
	if err != nil {
		log.Printf("Failed to read TCP DNS response: %v", err)
		return
	}

	// Parse DNS response
	resp := new(dns.Msg)
	err = resp.Unpack(response[:n])
	if err != nil {
		log.Printf("Failed to unpack DNS response: %v", err)
		return
	}

	fmt.Printf("TCP test successful! Query ID: %d\n", resp.Id)
	printDNSResponse(resp)
}

func testUDP(proxyAddr, dnsServer, queryDomain string) {
	// Create SOCKS5 client
	client, err := socks5.NewClient(proxyAddr, "", "", 0, 60)
	if err != nil {
		log.Printf("Failed to create SOCKS5 client: %v", err)
		return
	}

	// Test UDP connection to DNS server
	conn, err := client.Dial("udp", dnsServer)
	if err != nil {
		log.Printf("Failed to dial UDP through SOCKS5: %v", err)
		return
	}
	defer conn.Close()

	// Create DNS query using miekg/dns
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(queryDomain), dns.TypeA)
	m.Id = 54321 // Set a different recognizable ID

	// Pack the DNS message
	data, err := m.Pack()
	if err != nil {
		log.Printf("Failed to pack DNS query: %v", err)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		log.Printf("Failed to send UDP DNS query: %v", err)
		return
	}

	// Read response
	response := make([]byte, 512)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		log.Printf("Failed to read UDP DNS response: %v", err)
		return
	}

	// Parse DNS response
	resp := new(dns.Msg)
	err = resp.Unpack(response[:n])
	if err != nil {
		log.Printf("Failed to unpack DNS response: %v", err)
		return
	}

	fmt.Printf("UDP test successful! Query ID: %d\n", resp.Id)
	printDNSResponse(resp)
}

func testMultipleDNS(proxyAddr, queryDomain string) {
	dnsServers := []string{
		"8.8.8.8:53",        // Google
		"1.1.1.1:53",        // Cloudflare
		"208.67.222.222:53", // OpenDNS
	}

	for _, server := range dnsServers {
		fmt.Printf("\nTesting with DNS server: %s\n", server)
		testUDP(proxyAddr, server, queryDomain)
	}
}

func printDNSResponse(resp *dns.Msg) {
	fmt.Printf("Response Code: %s\n", dns.RcodeToString[resp.Rcode])
	fmt.Printf("Authoritative: %v\n", resp.Authoritative)
	fmt.Printf("Truncated: %v\n", resp.Truncated)
	fmt.Printf("Recursion Desired: %v\n", resp.RecursionDesired)
	fmt.Printf("Recursion Available: %v\n", resp.RecursionAvailable)

	if len(resp.Question) > 0 {
		fmt.Printf("Question: %s %s\n", resp.Question[0].Name, dns.TypeToString[resp.Question[0].Qtype])
	}

	if len(resp.Answer) > 0 {
		fmt.Printf("Answers (%d):\n", len(resp.Answer))
		for i, rr := range resp.Answer {
			fmt.Printf("  %d: %s\n", i+1, rr.String())
		}
	} else {
		fmt.Println("No answers in response")
	}

	if len(resp.Ns) > 0 {
		fmt.Printf("Authority (%d):\n", len(resp.Ns))
		for i, rr := range resp.Ns {
			fmt.Printf("  %d: %s\n", i+1, rr.String())
		}
	}

	if len(resp.Extra) > 0 {
		fmt.Printf("Additional (%d):\n", len(resp.Extra))
		for i, rr := range resp.Extra {
			fmt.Printf("  %d: %s\n", i+1, rr.String())
		}
	}
}
