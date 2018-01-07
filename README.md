# Spoof Source IP

Tool to play with spoofing the source IP on a UDP datagram.  It's written
in Golang using the gopacket library and, hopefully, provides a useful
example for creating packets using gopacket.

## Installation

#### Dependencies
To build the binaries from source, you'll need Golang installed and your
GOPATH environment variable configured.  I developed the code using
Golang 1.9.

gopacket, the library used from Google, is not written in pure
Golang.  It's uses `libpcap-dev` (package name on Linux systems) and
provides bindings using cgo which requires a C compiler.  I'm told the
installation needed to build gopacket projects on Windows can be
complicated, but it's fairly easy on Linux and macOS.

#### Permissions
`udpspoof` creates raw packets which requires special permissions. If
you don't wish to use `sudo` or run `udpspoof` as root, you can
assign the needed capability directly to the executable on Linux
systems:
```
$ sudo setcap 'CAP_NET_RAW+eip' $GOPATH/bin/udpspoof
```
On macOS, if you installed Wireshark, your user was most likely already
granted the needed permissions to run `udpspoof` without sudo.

`udpreceiver` does not use gopacket and requires no special permissions
to run on ports >= 1024.

#### Download/install
Install `udpreceiver` and `udpspoof`:
```
$ go get -u github.com/dimalinux/spoofsourceip/...
```
Only build/install the receiver (which does not depend on libpcap-dev):
```
go get -u github.com/dimalinux/spoofsourceip/udpreceiver
```
Only build/install the sender:
```
go get -u github.com/dimalinux/spoofsourceip/udpspoof
```

The binaries will be in $GOPATH/bin/ and the source in
$GOPATH/src/github.com/dimalinux/spoofsourceip.


## UDP Receiver (udpreceiver)
You can test if your spoofed datagrams are getting through with
`udpreceiver`.  It does not depend on `libpcap-dev` and is easy to build
and run anywhere.  By default, udpreceiver listens for UDP datagrams on
port 8888 for any local IP, but this can be modified with the flag
`--listen IPv4:port`.  For IPv6, use `--listen [IPv6]:port`.  To listen
on a specific port from any local IP address, use `--listen :port`.

You can sanity check the receiver using nmap:
```
$ sudo nmap -sU -p 8888 --data-length 9 PUT_IP_ADDRESS_HERE
```
In response to the above command, udpreceiver will show output like this:
```
2018/01/01 11:28:59 Datagram received fromIP=192.168.0.101 fromPort=44258 payload=0x1d0fda75758fb8e2ba
```

## UDP Spoofer (udpspoof)

`udpspoof` injects a UDP packet at the Ethernet frame level on the interface
specified by the --interface (-i) flag.  For `udpspoof` to work, the
destination MAC needs to be different from the MAC address of the interface
performing the injection.  Either use 2 hosts, or a single host with
multiple interfaces (e.g. a host with both wired and wireless interfaces).

You cannot use the loopback interface for testing, as the loopback
interface operates above the ethernet frame level and has no MAC address.

The source MAC and source IP are both spoofable.

View the full options list with the `--help` flag:
```
$ udpspoof --help
Usage of udpspoof:
  -i, --interface string     Network interface for packet injection (default "en0")
      --source-mac string    MAC address in the Ethernet frame source (default is not spoofed)
      --source-ip string     Sending address in the IP header (default is not spoofed)
      --source-port uint16   UDP Source port (default 9999)
      --dest-mac string      MAC address of the destination IP or gateway (required)
      --dest-ip string       Destination IP address (required)
      --dest-port uint16     Destination port (default 8888)
  -p, --payload string       UDP payload specified in hex (default "DEADBEEF")
```

### Full example: Receive DNS response on different host than request

Start the UDP receiver on a 2nd host which will receive a DNS
response from Google's DNS server.  The default port of 8888 is fine
for our needs:
```
host2$ udpreceiver
2018/01/01 11:35:42 Listening on [::]:8888
```

I've pre-captured the UDP payload of a DNS query looking up
"www.google.com" below.  (If you want a different query, it's easy
to capture your own in Wireshark.)  We'll send that payload to 8.8.8.8
(Google's public DNS server), but set the source IP and port to match
our second host above that is running `udpreceiver`.  Since the
destination IP (8.8.8.8) is not on the local network, we set the
destination hardware MAC address to the local network MAC on our
router-gateway.

On macOS and Linux, you can use the `arp` command to retrieve the MAC
address of your gateway (or any local peer by IP):
```
$ arp 45.63.20.1
Address             HWtype  HWaddress           Flags Mask        Iface
45.63.20.1          ether   fe:00:01:4a:6f:91   C                 ens3
```

Run the DNS query on host1 with the source IP/port set to the values for
the UDP receiver on host2.  The destination IP, 8.8.8.8 (Google's public
DNS server), is not on your local network, so set the destination MAC
address to your gateway router's LAN MAC address.

```
host1 $ udpspoof -i LOCAL_NETWORK_INTERFACE \
           --source-ip HOST2_IP --source-port 8888
           --dest-ip 8.8.8.8 --dest-port 53
           --dest-mac YOUR_GATEWAY_ROUTERS_LAN_MAC_ADDR
           --payload 519f012000010000000000010377777706676f6f676c6503636f6d00000100010000291000000000000000
```

After issuing the DNS query from host1, host2, will get a response like this
one.
```
2018/01/01 11:38:24 Datagram received from=8.8.8.8:53 truncated=false payloadHex=519f818000010001000000010377777706676f6
f676c6503636f6d0000010001c00c00010001000000330004acd902e40000290200000000000000
```
Note 1: Your response payload won't look the same as above since, among
other reasons, the IP address you receive for www.google.com will be
different.

Note 2: In most cases, the above example will work even if host1 and
host 2 are sitting behind a NAT firewall.  The source IP will be
replaced by the NAT router's IP before the query is sent to Google.
When Google sends the DNS response, the NAT router will forward it to
the forged source IP of the original DNS request.
