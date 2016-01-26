// Scans defined IP range and returns list of active devices, rtt, and DNS names.
// Started life based on mping (https://github.com/mhusmann/mping).
// Logging can be eliminated by redirecting stderr to /dev/null
package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	fastping "github.com/tatsushid/go-fastping"
)

// maxRTT is timeout for each ping
const maxRTT = time.Second
const numPing = 5

// Change constants to determine the range to be scanned.
const startIPString = "192.168.0.1"
const endIPString = "192.168.0.254"

type resultData struct {
	PingResult string
	HostResult string
}

var ips []resultData
var baseRTT time.Duration

// Utility functions
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func roundDuration(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

// Functions and types needed to support sorting the results

func (d resultData) getIP() net.IP {
	parts := strings.SplitN(d.PingResult, "\t", 2)
	dIP := net.ParseIP(parts[0])
	if dIP == nil {
		log.Fatal("parts[0],", parts[0], ", is not a valid IP address")
	}
	return dIP
}

type byIP []resultData

func (device byIP) Len() int      { return len(device) }
func (device byIP) Swap(i, j int) { device[i], device[j] = device[j], device[i] }
func (device byIP) Less(i, j int) bool {
	iIP := device[i].getIP()
	jIP := device[j].getIP()
	return bytes.Compare(iIP, jIP) < 0
}

func main() {
	useUDP := false
	log.Println(": Program started")

	startIP := net.ParseIP(startIPString)
	if startIP == nil {
		log.Fatal("startIPString,", startIPString, ", is not a valid IP address")
	}
	endIP := net.ParseIP(endIPString)
	if endIP == nil {
		log.Fatal("endIPString,", endIPString, ", is not a valid IP address")
	}

	netProto := "ip4:icmp"
	if strings.Index(startIPString, ":") != -1 {
		netProto = "ip6:ipv6-icmp"
	}

	p := fastping.NewPinger()
	p.MaxRTT = maxRTT
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		var device resultData
		device.PingResult = addr.String() + "\t" + roundDuration(baseRTT+rtt, time.Millisecond).String()
		ips = append(ips, device)
		p.RemoveIPAddr(addr)
	}

	currentIP := make(net.IP, len(startIP))
	for copy(currentIP, startIP); bytes.Compare(currentIP, endIP) <= 0; inc(currentIP) {
		ra, err := net.ResolveIPAddr(netProto, currentIP.String())
		if err != nil {
			log.Fatal(err)
		}
		p.AddIPAddr(ra)
	}

	if useUDP {
		p.Network("udp")
	}

	for index := 0; index < numPing; index++ {
		baseRTT = time.Duration(index) * maxRTT
		err := p.Run()
		if err != nil {
			log.Fatal("Pinger returns error: ", err)
		}
	}

	log.Println(": Scan complete")

	fmt.Println()
	fmt.Printf("%d devices found\n", len(ips))
	fmt.Println()

	// Query DNS for the name of each device found by the ping scan
	var ipAndTime []string
	var hostname string
	for index, ip := range ips {
		ipAndTime = strings.SplitN(ip.PingResult, "\t", 2)
		hosts, err := net.LookupAddr(ipAndTime[0])
		if err != nil {
			hostname = "Error: " + err.Error()
		} else {
			hostname = strings.Join(hosts, ", ")
		}
		ips[index].HostResult = hostname
	}

	log.Println(": DNS complete")
	sort.Sort(byIP(ips))
	log.Println(": Sort complete")

	for _, ip := range ips {
		fmt.Printf("%-25s\t--> %s\n", ip.PingResult, ip.HostResult)
	}

	fmt.Println()
	log.Println(": Program complete")
}
