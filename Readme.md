# ipscan - scan IP range and report IP, round-trip time (rtt), and DNS name for active devices

This program is based on [mping](https://github.com/mhusmann/mping) by [mhusmann](https://github.com/mhusmann).
It has been almost entirely re-written as I have learned more about Go and its concurrency patterns.

```
Usage: ipscan [-quv] [-n value] [-t value] startIP endIP
 -n, --count=value  max number of pings per target
 -q, --quiet        only display host data
 -t, --rtt=value    max RTT for each ping
 -u, --udp          use UDP instead of ICMP
 -v, --debug        print additional messages
 startIP, endIP     endpoints of scan (inclusive)
```

## To Do

- [x] Move range specification to command line argument(s)
- [x] Convert use of **fping** and **host** to native Go routines
- [x] Add support for IPv6 (untested)
- [ ] Modify text strings to facilitate internationalization
- [x] Improve performance
- [x] Improve robustness of scan
