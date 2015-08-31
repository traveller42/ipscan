// Scans defined IP range and returns list of active devices, rtt, and DNS names
// Started life based on mping (https://github.com/mhusmann/mping)
package main

import (
  "bufio"
  "fmt"
  "log"
  "os/exec"
  "sort"
  "strconv"
  "strings"
)

type Data struct {
  PingResult string
  HostResult string
}

func (d Data) Octets() []int {
  parts := strings.SplitN(d.PingResult, " ", 2)
  octetStrings := strings.SplitN(parts[0], ".", 4)
  octets := make([]int, 0)
  for _, octetString := range(octetStrings) {
    octetInt, _ := strconv.Atoi(octetString)
    octets = append(octets, octetInt)
  }
  return octets
}

type ByIP []Data

func (device ByIP) Len() int           { return len(device) }
func (device ByIP) Swap(i, j int)      { device[i], device[j] = device[j], device[i] }
func (device ByIP) Less(i, j int) bool {
  Ioctets := device[i].Octets()
  Joctets := device[j].Octets()
  for iter := 0; iter < 4; iter++ {
    switch {
      case Ioctets[iter] < Joctets[iter]:
        return true
      case Ioctets[iter] > Joctets[iter]:
        return false
    }
  }
  return false
}

const startIp = "192.168.0.1"
const endIp = "192.168.0.254"

func main() {
  log.Println(": Program started")

  cmd := exec.Command("fping", "-gae", startIp, endIp + " 2>/dev/null")
  cmdReader, err := cmd.StdoutPipe()
  if err != nil {
    log.Fatal("Error creating StdoutPipe for Cmd", err)
  }
  
  ips := make([]Data, 0)
  scanner := bufio.NewScanner(cmdReader)
  go func() {
    var device Data
    for scanner.Scan() {
      device.PingResult = scanner.Text()
      ips = append(ips, device)
    }
  }()
  
  err = cmd.Start()
  if err != nil {
    log.Fatal("Error starting Cmd", err)
  }
  log.Println(": Scan started")
  
  err = cmd.Wait()
  if err != nil && err.Error() != "exit status 1" {
    log.Fatal("Error waiting for Cmd", err)
  }
  log.Println(": Scan complete")

  fmt.Println()
  fmt.Printf("%d devices found\n", len(ips))
  fmt.Println()

  var ipAndTime []string
  var out1 []byte
  var out2 []string
  var hostname string
  for index, ip := range(ips) {
    ipAndTime = strings.SplitN(ip.PingResult, " ", 2)
    out1, _ = exec.Command("host", ipAndTime[0]).Output()
    // extract hostname from result
    out2 = strings.SplitAfterN(string(out1), "pointer ", 2)
    switch len(out2) {
    case 0:
      hostname = "<undefined>"
    case 1:
      hostname = out2[0]
    case 2:
      hostname = out2[1]
    default:
      log.Fatalf("Logic Error-> ip: %s result: %s", ipAndTime[0], string(out1))
    }
    ips[index].HostResult = hostname
  }

  log.Println(": DNS complete")
  sort.Sort(ByIP(ips))
  log.Println(": Sort complete")
  
  for _, ip := range(ips) {
    fmt.Printf("%-25s --> %s", ip.PingResult, ip.HostResult)
  }
  
  fmt.Println()
  log.Println(": Program complete")
}
