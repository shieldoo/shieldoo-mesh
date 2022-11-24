package main

import (
	"encoding/hex"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/go-ping/ping"
	"github.com/jackpal/gateway"
)

func NetutilsPing(ip string) bool {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		log.Debug("ping error: ", err)
		return false
	} else {
		pinger.Count = 2
		pinger.Timeout = time.Millisecond * 500
		pinger.Interval = time.Millisecond * 500
		pinger.SetPrivileged(true)
		err = pinger.Run() // Blocks until finished.
		if err != nil {
			return false
		} else {
			stats := pinger.Statistics()
			if stats.PacketsRecv > 0 {
				return true
			} else {
				return false
			}
		}
	}
}

func NetutilsGWDiscover() string {
	ret := ""
	dg, err := gateway.DiscoverGateway()
	if err != nil {
		log.Error("Cannot get default gateway for wstunnel: ", err)
	} else {
		ret = dg.String()
		log.Debug("System default gateway is: ", ret)
	}
	return ret
}

func NetutilsResolveDNS(fqdn string) ([]string, error) {
	fqdn = strings.TrimSpace(fqdn)
	log.Debug("dns resolve for: ", fqdn)
	ips, err := net.LookupIP(fqdn)
	if err != nil {
		log.Error("dns A resolve error: ", err)
		return []string{}, err
	}
	log.Debug("dns A: ", ips)
	var ret []string
	for _, i := range ips {
		ret = append(ret, i.String())
	}
	return ret, nil
}

type NetutilsCidrRange struct {
	FromIP net.IP
	ToIP   net.IP
}

func (m *NetutilsCidrRange) HEXString() string {
	return hex.EncodeToString(m.FromIP.To4()) + "-" + hex.EncodeToString(m.ToIP.To4())
}

func netutilsCidrFromStr(s string) net.IP { return net.ParseIP(s) }

func netutilsCidrAddIP(ip net.IP, inc int) net.IP {
	i := ip.To4()
	v := uint(i[0])<<24 + uint(i[1])<<16 + uint(i[2])<<8 + uint(i[3])
	if inc < 0 {
		inc = inc * -1
		v -= uint(inc)
	} else {
		v += uint(inc)
	}
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}

func netutilsCidrAddIPToRange(ip string, rng []NetutilsCidrRange) []NetutilsCidrRange {
	sort.Slice(rng, func(i, j int) bool {
		return rng[i].HEXString() < rng[j].HEXString()
	})
	ipr := NetutilsCidrRange{FromIP: netutilsCidrFromStr(ip), ToIP: netutilsCidrFromStr(ip)}
	var ret []NetutilsCidrRange
	for idx, i := range rng {
		if i.HEXString() > ipr.HEXString() && idx > 0 {
			ipr.ToIP = rng[idx-1].ToIP
			ipr.FromIP = netutilsCidrAddIP(netutilsCidrFromStr(ip), 1)
			rng[idx-1].ToIP = netutilsCidrAddIP(netutilsCidrFromStr(ip), -1)
			break
		}
	}
	ret = append(rng, ipr)
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].HEXString() < ret[j].HEXString()
	})
	return ret
}

func NetutilsGenerateRoutes(ipExceptions []string) []string {
	var ret []string
	arr := []NetutilsCidrRange{
		{FromIP: netutilsCidrFromStr("0.0.0.0"), ToIP: netutilsCidrFromStr("9.255.255.255")},
		{FromIP: netutilsCidrFromStr("192.169.0.0"), ToIP: netutilsCidrFromStr("255.255.255.255")},
		{FromIP: netutilsCidrFromStr("172.32.0.0"), ToIP: netutilsCidrFromStr("192.167.255.255")},
		{FromIP: netutilsCidrFromStr("11.0.0.0"), ToIP: netutilsCidrFromStr("172.15.255.255")},
	}

	for _, i := range ipExceptions {
		arr = netutilsCidrAddIPToRange(i, arr)
	}

	for _, a := range arr {
		for _, r := range netutilsCidrConvertIP(a.FromIP, a.ToIP) {
			ret = append(ret, r.String())
		}
	}
	return ret
}

var netutilsCidrAll = net.ParseIP("255.255.255.255").To4()

func netutilsCidrConvertIP(a1, a2 net.IP) (r []*net.IPNet) {
	maxLen := 32
	a1 = a1.To4()
	a2 = a2.To4()
	for netutilsCidrCompare(a1, a2) <= 0 {
		l := 32
		for l > 0 {
			m := net.CIDRMask(l-1, maxLen)
			if netutilsCidrCompare(a1, netutilsCidrFirst(a1, m)) != 0 || netutilsCidrCompare(netutilsCidrLast(a1, m), a2) > 0 {
				break
			}
			l--
		}
		r = append(r, &net.IPNet{IP: a1, Mask: net.CIDRMask(l, maxLen)})
		a1 = netutilsCidrLast(a1, net.CIDRMask(l, maxLen))
		if netutilsCidrCompare(a1, netutilsCidrAll) == 0 {
			break
		}
		a1 = netutilsCidrNext(a1)
	}
	return r
}

func netutilsCidrCompare(ip1, ip2 net.IP) int {
	l := len(ip1)
	for i := 0; i < l; i++ {
		if ip1[i] == ip2[i] {
			continue
		}
		if ip1[i] < ip2[i] {
			return -1
		}
		return 1
	}
	return 0
}

func netutilsCidrFirst(ip net.IP, mask net.IPMask) net.IP {
	return ip.Mask(mask)
}

func netutilsCidrLast(ip net.IP, mask net.IPMask) net.IP {
	n := len(ip)
	ret := make(net.IP, n)
	for i := 0; i < n; i++ {
		ret[i] = ip[i] | ^mask[i]
	}
	return ret
}

func netutilsCidrNext(ip net.IP) net.IP {
	n := len(ip)
	ret := make(net.IP, n)
	isCopy := false
	for n > 0 {
		n--
		if isCopy {
			ret[n] = ip[n]
			continue
		}
		if ip[n] < 255 {
			ret[n] = ip[n] + 1
			isCopy = true
			continue
		}
		ret[n] = 0
	}
	return ret
}
