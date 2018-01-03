package main

import (
	"encoding/hex"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
)

type cliOptions struct {
	networkIface *net.Interface
	udpFrameOptions
}

const (
	defaultInterfaceName = "en0"
	defaultPayloadHex    = "DEADBEEF"
	defaultSourcePort    = 9999
	defaultDestPort      = 8888 // matches UDP receiver's default UDP listen port
)

func parseCommandLineArgs() *cliOptions {

	cliOps := &cliOptions{}
	var sourceMACArg, sourceIPArg string
	var destMACArg, destIPArg string
	var interfaceArg, payloadHexArg string

	//
	// Setup flags and parse
	//
	pflag.CommandLine.SortFlags = false
	pflag.StringVarP(&interfaceArg, "interface", "i", defaultInterfaceName,
		"Network interface for packet injection")

	pflag.StringVar(&sourceMACArg, "source-mac", "",
		"MAC address in the Ethernet frame source (default is not spoofed)")
	pflag.StringVar(&sourceIPArg, "source-ip", "",
		"Sending address in the IP header (default is not spoofed)")
	pflag.Uint16Var(&cliOps.sourcePort, "source-port", defaultSourcePort,
		"UDP Source port")

	pflag.StringVar(&destMACArg, "dest-mac", "",
		"MAC address of the destination IP or gateway (required)")
	pflag.StringVar(&destIPArg, "dest-ip", "",
		"Destination IP address (required)")
	pflag.Uint16Var(&cliOps.destPort, "dest-port", defaultDestPort,
		"Destination port")

	pflag.StringVarP(&payloadHexArg, "payload", "p", defaultPayloadHex,
		"UDP payload specified in hex")
	pflag.Parse()

	//
	//  Validate/process parsed args
	//
	var err error
	if interfaceArg == "" {
		log.Fatal("Network interface (--interface/-i) is required. Example: udpspoof -i eth0 ...")
	}
	if cliOps.networkIface, err = net.InterfaceByName(interfaceArg); err != nil {
		log.Fatalf("Invalid network interface name '%s': %s", interfaceArg, err)
	}
	if cliOps.networkIface.HardwareAddr == nil {
		log.Fatalf("Network interface '%s' has no MAC address", interfaceArg)
	}

	if destMACArg == "" {
		log.Fatal("Destination MAC (--dest-mac) is required")
	}
	destMACArg = standardizeMACFormat(destMACArg)
	if destMACArg == cliOps.networkIface.HardwareAddr.String() {
		log.Fatal("Destination MAC must be different from the injecting network interface's MAC")
	}
	if cliOps.destMac, err = net.ParseMAC(destMACArg); err != nil {
		log.Fatalf("Invalid destination MAC '%s': %s", destMACArg, err)
	}

	if sourceMACArg == "" {
		// default the source MAC to its legitimate value if unset
		cliOps.sourceMac = cliOps.networkIface.HardwareAddr
	} else {
		sourceMACArg = standardizeMACFormat(sourceMACArg)
		if cliOps.sourceMac, err = net.ParseMAC(sourceMACArg); err != nil {
			log.Fatalf("Invalid source MAC '%s': %s", sourceMACArg, err)
		}
	}

	if destIPArg == "" {
		log.Fatal("Destination IP (--dest-ip) is required")
	}
	if cliOps.destIP = net.ParseIP(destIPArg); cliOps.destIP == nil {
		log.Fatalf("Invalid dest IP '%s'", destIPArg)
	}
	cliOps.isIPv6 = strings.Contains(destIPArg, ":")

	if sourceIPArg == "" {
		// default the source IP to its legitimate value
		sourceIPArg = getInterfaceIP(cliOps.networkIface, cliOps.isIPv6)
		if sourceIPArg == "" {
			log.Fatalf("Unable to find compatible IPv6=%t source IP", cliOps.isIPv6)
		}
	} else {
		sourceIsIPv6 := strings.Contains(sourceIPArg, ":")
		if sourceIsIPv6 != cliOps.isIPv6 {
			log.Fatal("Source and destination IP must be of same type (IPv4 or IPv6")
		}
	}
	if cliOps.sourceIP = net.ParseIP(sourceIPArg); cliOps.sourceIP == nil {
		log.Fatalf("Invalid source IP '%s'", sourceIPArg)
	}

	// don't give an error if the user includes a "0x" prefix.
	payloadHexArg = strings.TrimPrefix(payloadHexArg, "0x")
	if cliOps.payloadBytes, err = hex.DecodeString(payloadHexArg); err != nil {
		log.Fatalf("Unable to decode hex payload: %s", err)
	}

	return cliOps
}

// standardizeMACFormat fixes dash-separated MAC addresses from Windows ipconfig
// and macOS arp results which don't include leading zeros (:9: instead of :09:)
func standardizeMACFormat(macAddr string) string {
	macAddr = strings.Replace(macAddr, "-", ":", -1)
	return regexp.MustCompile(`(\b)(\d)(\b)`).ReplaceAllString(macAddr, "${1}0${2}${3}")
}

// getInterfaceIP searches addresses of the passed-in interface for the first address
// matching the requested type (IPv4 or IPv6) and returns it's value as a string. If
// no compatible address is found, the empty string is returned.
func getInterfaceIP(iface *net.Interface, useIPv6 bool) string {
	if addresses, err := iface.Addrs(); err == nil {
		for _, addr := range addresses {
			addrStr := addr.String()
			isIPv6 := strings.Contains(addrStr, ":")
			if isIPv6 == useIPv6 {
				return addrStr[0:strings.Index(addrStr, "/")]
			}
		}
	}
	return ""
}
