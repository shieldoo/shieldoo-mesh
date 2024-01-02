package main

import (
	"net"
	"reflect"
	"testing"
)

func checkIPInCIDRs(t *testing.T, ip string, cidr []string) bool {
	ipParsed := net.ParseIP(ip)
	for _, i := range cidr {
		_, ipNet, err := net.ParseCIDR(i)
		if err != nil {
			t.Logf("Error parsing CIDR: %v\n", err)
			return false
		}
		if ipNet.Contains(ipParsed) {
			return true
		}
	}
	return false
}

func parseCIDRToFromTo(t *testing.T, cidr string) (string, string) {
	// Parse the CIDR notation
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Logf("Error parsing CIDR: %v\n", err)
		return "", ""
	}
	// Convert the parsed IP to a 4-byte representation if it's IPv4
	startIP := ip.To4()
	if startIP == nil {
		startIP = ip
	}

	// Calculate the ending IP address
	endIP := make(net.IP, len(startIP))
	for i := range startIP {
		endIP[i] = startIP[i] | ^ipNet.Mask[i]
	}

	return startIP.String(), endIP.String()
}

func TestNetutilsGenerateRoutesEmpty(t *testing.T) {
	ipExceptions := []string{}
	expectedRoutes := []string{
		"0.0.0.0/5",
		"8.0.0.0/7",
		"11.0.0.0/8",
		"12.0.0.0/6",
		"16.0.0.0/4",
		"32.0.0.0/3",
		"64.0.0.0/2",
		"128.0.0.0/3",
		"160.0.0.0/5",
		"168.0.0.0/6",
		"172.0.0.0/12",
		"172.32.0.0/11",
		"172.64.0.0/10",
		"172.128.0.0/9",
		"173.0.0.0/8",
		"174.0.0.0/7",
		"176.0.0.0/4",
		"192.0.0.0/9",
		"192.128.0.0/11",
		"192.160.0.0/13",
		"192.169.0.0/16",
		"192.170.0.0/15",
		"192.172.0.0/14",
		"192.176.0.0/12",
		"192.192.0.0/10",
		"193.0.0.0/8",
		"194.0.0.0/7",
		"196.0.0.0/6",
		"200.0.0.0/5",
		"208.0.0.0/4",
		"224.0.0.0/3",
	}

	routes := NetutilsGenerateRoutes(ipExceptions)

	// print result
	for _, i := range routes {
		// also decode CIDR to from IP - to IP
		fromIP, toIP := parseCIDRToFromTo(t, i)
		t.Logf("Route: %s (%s - %s)", i, fromIP, toIP)
	}

	if !reflect.DeepEqual(routes, expectedRoutes) {
		t.Errorf("Generated routes do not match expected routes")
	}
}

func TestNetutilsGenerateRoutesOneA(t *testing.T) {
	ipExceptions := []string{"52.18.35.215"}

	routes := NetutilsGenerateRoutes(ipExceptions)

	// print result
	for _, i := range routes {
		// also decode CIDR to from IP - to IP
		fromIP, toIP := parseCIDRToFromTo(t, i)
		t.Logf("Route: %s (%s - %s)", i, fromIP, toIP)
	}

	for _, i := range ipExceptions {
		if checkIPInCIDRs(t, i, routes) {
			t.Errorf("IP %s not in generated routes, but routes are exceptions!", i)
		}
	}
}

func TestNetutilsGenerateRoutesOneB(t *testing.T) {
	ipExceptions := []string{"195.201.144.201"}

	routes := NetutilsGenerateRoutes(ipExceptions)

	// print result
	for _, i := range routes {
		// also decode CIDR to from IP - to IP
		fromIP, toIP := parseCIDRToFromTo(t, i)
		t.Logf("Route: %s (%s - %s)", i, fromIP, toIP)
	}

	for _, i := range ipExceptions {
		if checkIPInCIDRs(t, i, routes) {
			t.Errorf("IP %s not in generated routes, but routes are exceptions!", i)
		}
	}
}

func TestNetutilsGenerateRoutesArrayA(t *testing.T) {
	ipExceptions := []string{"195.201.144.201", "128.140.104.51", "49.13.149.80", "142.132.169.89"}

	routes := NetutilsGenerateRoutes(ipExceptions)

	// print result
	for _, i := range routes {
		// also decode CIDR to from IP - to IP
		fromIP, toIP := parseCIDRToFromTo(t, i)
		t.Logf("Route: %s (%s - %s)", i, fromIP, toIP)
	}

	for _, i := range ipExceptions {
		if checkIPInCIDRs(t, i, routes) {
			t.Errorf("IP %s not in generated routes, but routes are exceptions!", i)
		}
	}
}
