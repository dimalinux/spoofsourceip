package main

import (
	"encoding/hex"
	"log"
	"net"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/spf13/pflag"
)

var interfaceName = "en0"
var sourceIPStr = "8.8.8.8"
var sourceIP net.IP
var sourcePort uint16 = 9999
var sourceIface *net.Interface
var usingIPv6 bool
var destIPStr string
var destIP net.IP
var destPort uint16 = 8888
var destMacStr string
var destMacHw net.HardwareAddr
var payloadHex = "DEADBEEF"
var payloadBytes []byte

func init() {
	pflag.StringVarP(&interfaceName, "interface", "i", interfaceName,
		"Local network device name that will send the packet")
	pflag.StringVar(&sourceIPStr, "source-ip", sourceIPStr,
		"[Spoofed] Sending address in the packet's IP header")
	pflag.Uint16Var(&sourcePort, "source-port", sourcePort,
		"[Spoofed] Source port")
	pflag.StringVar(&destMacStr, "dest-mac", destMacStr,
		"MAC address of the destination IP or gateway (default uses the source MAC)")
	pflag.StringVar(&destIPStr, "dest-ip", destIPStr,
		"Destination IP address (default uses an address on the local --interface device)")
	pflag.Uint16Var(&destPort, "dest-port", destPort,
		"Destination port")
	pflag.StringVarP(&payloadHex, "payload", "p", payloadHex,
		"UDP payload specified in hex with no leading '0x'")
	pflag.Parse()

	var err error
	if sourceIP = net.ParseIP(sourceIPStr); sourceIP == nil {
		log.Fatalf("Invalid source IP '%s'", sourceIPStr)
	}
	usingIPv6 = strings.Contains(sourceIPStr, ":")
	if sourceIface, err = net.InterfaceByName(interfaceName); err != nil {
		log.Fatalf("Invalid source interface name '%s': %s", interfaceName, err)
	}
	if sourceIface.HardwareAddr == nil {
		log.Fatalf("Source interface '%s' has no MAC address", interfaceName)
	}

	if destIPStr == "" {
		if destIPStr = getInterfaceIP(sourceIface, usingIPv6); destIPStr == "" {
			log.Fatalf("Unable to find IP address on interface '%s'", sourceIface.Name)
		}
	}
	if destIP = net.ParseIP(destIPStr); destIP == nil {
		log.Fatalf("Invalid dest IP '%s'", destIPStr)
	}
	if destMacStr == "" {
		destMacStr = sourceIface.HardwareAddr.String()
	}
	if destMacHw, err = net.ParseMAC(destMacStr); err != nil {
		log.Fatalf("Invalid destination MAC: '%s'", destMacStr)
	}
	if payloadBytes, err = hex.DecodeString(payloadHex); err != nil {
		log.Fatalf("Unable to decode hex payload: %s", err)
	}
}

// Searches addresses of the passed-in interface for the first requested address type
// (IPv4 or IPv6) and returns it's value as a string.  If no compatible address is
// found, the empty string is returned.
func getInterfaceIP(iface *net.Interface, useIPv6 bool) string {
	if addresses, err := iface.Addrs(); err == nil {
		for _, addr := range addresses {
			addrStr := addr.String()
			isIPv6 := strings.Contains(addrStr, ":")
			if (isIPv6 && useIPv6) || (!isIPv6 && !useIPv6) {
				return addrStr[0:strings.Index(addrStr, "/")]
			}
		}
	}
	return ""
}

type serializableNetworkLayer interface {
	gopacket.NetworkLayer
	gopacket.SerializableLayer
}

// addrAsString returns a string with the format "ipv4:port" or "[ipv6]:port".
func addrAsString(ip net.IP, port uint16) string {
	return (&net.UDPAddr{IP:ip, Port: int(port)}).String()
}

func main() {
	log.Printf("Source IP / MAC: %s / %v", addrAsString(sourceIP, sourcePort), sourceIface.HardwareAddr)
	log.Printf("Dest   IP / MAC: %s / %v", addrAsString(destIP, destPort), destMacHw)

	handle, err := pcap.OpenLive(interfaceName, 1024, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	ethernetType := layers.EthernetTypeIPv4
	if usingIPv6 {
		ethernetType = layers.EthernetTypeIPv6
	}
	eth := &layers.Ethernet{
		SrcMAC:       sourceIface.HardwareAddr,
		DstMAC:       destMacHw,
		EthernetType: ethernetType,
	}
	var ip serializableNetworkLayer
	if !usingIPv6 {
		ip = &layers.IPv4{
			SrcIP:    sourceIP,
			DstIP:    destIP,
			Protocol: layers.IPProtocolUDP,
			Version:  4,
			TTL:      32,
		}
	} else {
		ip = &layers.IPv6{
			SrcIP:      sourceIP,
			DstIP:      destIP,
			NextHeader: layers.IPProtocolUDP,
			Version:    6,
			HopLimit:   32,
		}
		ip.LayerType()
	}

	udp := &layers.UDP{
		SrcPort: layers.UDPPort(sourcePort),
		DstPort: layers.UDPPort(destPort),
		// we configured "Length" and "Checksum" to be set for us
	}
	udp.SetNetworkLayerForChecksum(ip)
	err = gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payloadBytes))

	if err != nil {
		log.Fatal(err)
	}

	if err := handle.WritePacketData(buf.Bytes()); err != nil {
		log.Fatal(err)
	}
}
