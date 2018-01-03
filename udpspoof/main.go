package main

import (
	"log"
	"net"

	"github.com/google/gopacket/pcap"
)

// addrAsString returns a string with the format "ipv4:port" or "[ipv6]:port".
func addrAsString(ip net.IP, port uint16) string {
	return (&net.UDPAddr{IP: ip, Port: int(port)}).String()
}

func main() {
	cliOps := parseCommandLineArgs()

	log.Printf("Source IP / MAC: %s / %v", addrAsString(cliOps.sourceIP, cliOps.sourcePort), cliOps.sourceMac)
	log.Printf("Dest   IP / MAC: %s / %v", addrAsString(cliOps.destIP, cliOps.destPort), cliOps.destMac)

	handle, err := pcap.OpenLive(cliOps.networkIface.Name, 1024, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	frameBytes, err := createSerializedUDPFrame(cliOps.udpFrameOptions)
	if err != nil {
		log.Fatal(err)
	}

	if err := handle.WritePacketData(frameBytes); err != nil {
		log.Fatal(err)
	}
}
