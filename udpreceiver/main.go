package main

import (
	"encoding/hex"
	"log"
	"net"

	"github.com/spf13/pflag"
)

const maxPayload = 1040
var bindAddress = ":8888" // Default listen on all IPv4 and IPv6 addresses

func init() {
	pflag.StringVar(&bindAddress, "listen", bindAddress, "Local IP:port to listen for UDP packets on")
	pflag.Parse()
}

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", bindAddress)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Printf("Listening on %v", conn.LocalAddr())

	for {
		payloadData := make([]byte, maxPayload)
		sz, addr, err := conn.ReadFromUDP(payloadData)
		if err != nil {
			log.Fatal(err)
		}
		// This is the best we can do without using more complicated methods.  The downside is a
		// false positive as truncated when there is an exact fit at the maximum configured size.
		truncated := sz == maxPayload
		log.Printf("Datagram received from=%v truncated=%t payloadHex=%s",
			addr, truncated, hex.EncodeToString(payloadData[0:sz]))
	}
}
