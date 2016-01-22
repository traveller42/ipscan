// Scans defined IP range and returns list of active devices, rtt, and DNS names.
// Started life based on mping (https://github.com/mhusmann/mping).
// Currently dependent on the commands 'fping' and 'host' and a Bourne-ish shell environment.
// Logging can be eliminated by redirecting stderr to /dev/null
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Change constants to determine the range to be scanned.
const startIP = "192.168.0.1"
const endIP = "192.168.0.254"

type resultData struct {
	PingResult string
	HostResult string
}

// Functions and types needed to support sorting the results

func (d resultData) addrOctets() []int {
	parts := strings.SplitN(d.PingResult, " ", 2)
	octetStrings := strings.SplitN(parts[0], ".", 4)
	var octets []int
	for _, octetString := range octetStrings {
		octetInt, _ := strconv.Atoi(octetString)
		octets = append(octets, octetInt)
	}
	return octets
}

type byIP []resultData

func (device byIP) Len() int      { return len(device) }
func (device byIP) Swap(i, j int) { device[i], device[j] = device[j], device[i] }
func (device byIP) Less(i, j int) bool {
	Ioctets := device[i].addrOctets()
	Joctets := device[j].addrOctets()
	for iter := 0; iter < 4; iter++ {
		switch {
		case Ioctets[iter] < Joctets[iter]:
			return true
		case Ioctets[iter] > Joctets[iter]:
			return false
		}
	}
	// This function should have returned by now as there shoudn't be duplicate IPs
	// This final return is correct for the case where duplicate IPs are compared
	// i < j is false for the case i == j
	return false
}

func main() {
	log.Println(": Program started")

	// Configure an exec call to run fping in a subprocess and sent the output to a pipe
	cmd := exec.Command("fping", "-gae", startIP, endIP+" 2>/dev/null")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("Error creating StdoutPipe for Cmd", err)
	}

	// Create a goroutine to read the output from the pipe
	var ips []resultData
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		var device resultData
		for scanner.Scan() {
			device.PingResult = scanner.Text()
			ips = append(ips, device)
		}
	}()

	// Start the subprocess prepared above
	err = cmd.Start()
	if err != nil {
		log.Fatal("Error starting Cmd", err)
	}
	log.Println(": Scan started")

	// Wait for the subprocess to exit
	err = cmd.Wait()
	if err != nil && err.Error() != "exit status 1" {
		log.Fatal("Error waiting for Cmd", err)
	}
	log.Println(": Scan complete")

	fmt.Println()
	fmt.Printf("%d devices found\n", len(ips))
	fmt.Println()

	// Query DNS for the name of each device found by the ping scan
	var ipAndTime []string
	var hosts []string
	var hostname string
	for index, ip := range ips {
		ipAndTime = strings.SplitN(ip.PingResult, " ", 2)
		hosts, err = net.LookupAddr(ipAndTime[0])
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
		fmt.Printf("%-25s --> %s\n", ip.PingResult, ip.HostResult)
	}

	fmt.Println()
	log.Println(": Program complete")
}
