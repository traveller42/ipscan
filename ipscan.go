// Scans specified IP range and returns list of active devices, rtt, and DNS names.
// Started life based on mping (https://github.com/mhusmann/mping).
//
// Usage: ipscan [-quv] [-n value] [-t value] startIP endIP
//  -n, --count=value  max number of pings per target
//  -q, --quiet        only display host data
//  -t, --rtt=value    max RTT for each ping
//  -u, --udp          use UDP instead of ICMP
//  -v, --debug        print additional messages
//  startIP, endIP     endpoints of scan (inclusive)
package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	getopt "github.com/pborman/getopt"
	fastping "github.com/tatsushid/go-fastping"
)

// defaultMaxRTT is timeout for each ping
const defaultMaxRTT = time.Second
const defaultPingCount = 5

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
	// Uncomment the following lines if you need to time the options parsing
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	//log.Println(": Program started")

	// Configure Command Line Options
	var useUDP, quiet, debug bool
	var maxRTT time.Duration
	var numPing int
	getopt.BoolVarLong(&useUDP, "udp", 'u', "use UDP instead of ICMP")
	getopt.BoolVarLong(&quiet, "quiet", 'q', "only display host data")
	getopt.BoolVarLong(&debug, "debug", 'v', "print additional messages")
	maxRTT = defaultMaxRTT
	getopt.DurationVarLong(&maxRTT, "rtt", 't', "max RTT for each ping")
	numPing = defaultPingCount
	getopt.IntVarLong(&numPing, "count", 'n', "max number of pings per target")
	getopt.SetParameters("startIP endIP")
	getopt.Parse()

	// Verify arguments
	if getopt.NArgs() != 2 {
		log.Println("Incorrect number of arguments!")
		getopt.PrintUsage(os.Stderr)
		os.Exit(1)
	}
	startIPString := getopt.Arg(0)
	endIPString := getopt.Arg(1)

	// Test for incompatible options
	if quiet && debug {
		log.Println("`quiet` and `debug` are incompatible")
		getopt.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	if debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		log.Println(": Command Line Parsing complete")
	}

	// Convert to IP object
	startIP := net.ParseIP(startIPString)
	if startIP == nil {
		log.Fatal("Start IP,", startIPString, ", is not a valid IP address")
	}
	if debug {
		log.Println(": Start IP\t", startIPString)
	}
	endIP := net.ParseIP(endIPString)
	if endIP == nil {
		log.Fatal("End IP,", endIPString, ", is not a valid IP address")
	}
	if debug {
		log.Println(": End IP  \t", endIPString)
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

	if debug {
		log.Println(": Start Scan")
	}

	for index := 0; index < numPing; index++ {
		baseRTT = time.Duration(index) * maxRTT
		err := p.Run()
		if err != nil {
			log.Fatal("Pinger returns error: ", err)
		}
	}

	if debug {
		log.Println(": Scan complete")
	}

	if !quiet {
		fmt.Println()
		fmt.Printf("%d devices found\n", len(ips))
		fmt.Println()
	}

	if debug {
		log.Println(": Start Host Lookup")
	}

	// Query DNS for the name of each device found by the ping scan
	var ipAndTime []string
	var hostname string
	var wg sync.WaitGroup
	for index, ip := range ips {
		ipAndTime = strings.SplitN(ip.PingResult, "\t", 2)
		wg.Add(1)
		go func(ipString string, localIndex int) {
			hosts, err := net.LookupAddr(ipString)
			if err != nil {
				hostname = "Error: " + err.Error()
			} else {
				hostname = strings.Join(hosts, ", ")
			}
			ips[localIndex].HostResult = hostname
			wg.Done()
		}(ipAndTime[0], index)
	}
	wg.Wait()

	if debug {
		log.Println(": DNS complete")
	}

	sort.Sort(byIP(ips))

	if debug {
		log.Println(": Sort complete")
	}

	for _, ip := range ips {
		fmt.Printf("%-25s\t--> %s\n", ip.PingResult, ip.HostResult)
	}

	if !quiet {
		fmt.Println()
	}

	if debug {
		log.Println(": Program complete")
	}
}
