package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

var oids = map[string]string{
	"lldpRemSysName":  ".1.0.8802.1.1.2.1.4.1.1.9",
	"lldpRemPortId":   ".1.0.8802.1.1.2.1.4.1.1.7",
	"lldpRemPortDesc": ".1.0.8802.1.1.2.1.4.1.1.8",
	"ifDescr":         ".1.3.6.1.2.1.2.2.1.2",
	"ifName":          ".1.3.6.1.2.1.31.1.1.1.1",
}

type lldpEntry struct {
	LocalPortName  string
	RemotePortName string
	RemoteSysName  string
}

func main() {
	var ip = flag.String("ip", "", "IP address of target device")
	var community = flag.String("c", "public", "SNMP community")
	var format = flag.String("o", "csv", "Output format (csv, json)")
	var localPortType = flag.String("lt", "name", "port-id-subtype selection for local (name, desc)")
	var remotePortType = flag.String("rt", "id", "port-id-subtype selection for remote (id, desc)")
	var prune = flag.Bool("p", false, "do not output LLDP entry which has no remote info")

	flag.Parse()

	if net.ParseIP(*ip) == nil {
		log.Fatal("-ip argument is not set or invalid")
	}
	if *localPortType != "name" && *localPortType != "desc" {
		log.Fatal("-lt flag argument should be \"name\" or \"desc\"")
	}

	if *remotePortType != "id" && *remotePortType != "desc" {
		log.Fatal("-rt flag argument should be \"id\" or \"desc\" ")
	}

	target := &gosnmp.GoSNMP{
		Target:    *ip,
		Community: *community,
		Port:      161,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(1) * time.Second,
	}

	err := target.Connect()
	if err != nil {
		log.Fatal(err)
	}

	lldpEntries := map[string]*lldpEntry{}

	var results []gosnmp.SnmpPDU

	if *localPortType == "name" {
		results, err = target.BulkWalkAll(oids["ifName"])
	} else if *localPortType == "desc" {
		results, err = target.BulkWalkAll(oids["ifDescr"])
	}
	if err != nil || len(results) == 0 {
		log.Fatal("Failed to get local port information.")
	}

	for _, pdu := range results {
		localPortName := string(pdu.Value.([]uint8))
		if *localPortType == "name" {
			lldpEntries[pdu.Name[24:]] = &lldpEntry{LocalPortName: localPortName}
		} else if *localPortType == "desc" {
			lldpEntries[pdu.Name[21:]] = &lldpEntry{LocalPortName: localPortName}
		}
	}

	results, err = target.BulkWalkAll(oids["lldpRemSysName"])
	if err != nil || len(results) == 0 {
		log.Fatal("Failed to get remote system name.")
	}
	for _, pdu := range results {
		sysName := string(pdu.Value.([]uint8))
		lldpEntries[strings.Split(pdu.Name[26:], ".")[1]].RemoteSysName = sysName
	}

	if *remotePortType == "desc" {
		results, err = target.BulkWalkAll(oids["lldpRemPortDesc"])
	} else if *remotePortType == "id" {
		results, err = target.BulkWalkAll(oids["lldpRemPortId"])
	}
	if err != nil || len(results) == 0 {
		log.Fatal("Failed to get remote port information.")
	}

	for _, pdu := range results {
		remotePort := string(pdu.Value.([]uint8))
		lldpEntries[strings.Split(pdu.Name[26:], ".")[1]].RemotePortName = remotePort
	}

	if *prune {
		for key, lldp := range lldpEntries {
			if lldp.RemotePortName == "" && lldp.RemoteSysName == "" {
				delete(lldpEntries, key)
			}
		}
	}

	if *format == "json" {
		jsonString, err := json.Marshal(lldpEntries)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonString))
	} else if *format == "csv" {
		fmt.Println("Local,RemotePort,RemoteSysName")
		for _, lldp := range lldpEntries {
			fmt.Println(
				lldp.LocalPortName, ",",
				lldp.RemotePortName, ",",
				lldp.RemoteSysName,
			)
		}
	}

	os.Exit(0)

}
