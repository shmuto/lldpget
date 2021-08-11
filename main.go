package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

var oids = map[string]string{
	"lldpLocPortId":  ".1.0.8802.1.1.2.1.3.7.1.3",
	"lldpRemSysName": ".1.0.8802.1.1.2.1.4.1.1.9",
	"lldpRemPortId":  ".1.0.8802.1.1.2.1.4.1.1.7",
}

type lldpEntry struct {
	LocalPortId   string
	RemotePortId  string
	RemoteSysName string
}

func main() {
	var ip = flag.String("ip", "127.0.0.1", "IP address of target device")
	var community = flag.String("c", "public", "SNMP community")
	var format = flag.String("o", "csv", "Output format (csv, json)")

	flag.Parse()

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

	results, err := target.BulkWalkAll(oids["lldpLocPortId"])

	for _, pdu := range results {
		ifDescr := string(pdu.Value.([]uint8))
		lldpEntries[pdu.Name[26:]] = &lldpEntry{LocalPortId: ifDescr}
	}

	results, err = target.BulkWalkAll(oids["lldpRemSysName"])
	if err != nil {
		log.Fatal(err)
	}
	for _, pdu := range results {
		sysName := string(pdu.Value.([]uint8))
		lldpEntries[strings.Split(pdu.Name[26:], ".")[1]].RemoteSysName = sysName
	}

	results, err = target.BulkWalkAll(oids["lldpRemPortId"])
	if err != nil {
		log.Fatal(err)
	}
	for _, pdu := range results {
		remotePort := string(pdu.Value.([]uint8))
		lldpEntries[strings.Split(pdu.Name[26:], ".")[1]].RemotePortId = remotePort
	}

	if *format == "json" {
		jsonString, err := json.Marshal(lldpEntries)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonString))
		return
	} else if *format == "csv" {
		fmt.Println("Local,RemotePort,RemoteSysName")
		for _, lldp := range lldpEntries {
			if lldp.RemotePortId == "" && lldp.RemoteSysName == "" {
				continue
			}
			fmt.Println(
				lldp.LocalPortId, ",",
				lldp.RemotePortId, ",",
				lldp.RemoteSysName,
			)
		}
		return
	}

}
