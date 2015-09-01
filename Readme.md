# ipscan - scan IP range and report IP, round-trip time (rtt), and DNS name for active devices

This program is based on [mping](https://github.com/mhusmann/mping) by [mhusmann](https://github.com/mhusmann).
It has been almost entirely re-written as I have learned more about Go and its concurrency patterns.

Range endpoints are currently hard-coded as constants near the top of the source.

## To Do

- Move range specification to command line argument(s)
- Convert use of **fping** and **host** to native Go routines
- Add support for IPv6
- Modify text strings to facilitate internationalization
