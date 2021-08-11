package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

var oids = map[string]string{
	"lldpLocPortId":   ".1.0.8802.1.1.2.1.3.7.1.3",
	"lldpLocPortDesc": ".1.0.8802.1.1.2.1.3.7.1.4",
	"lldpRemSysName":  ".1.0.8802.1.1.2.1.4.1.1.9",
	"lldpRemPortId":   ".1.0.8802.1.1.2.1.4.1.1.7",
	"lldpRemPortDesc": ".1.0.8802.1.1.2.1.4.1.1.8",
}

type lldpEntry struct {
	LocalPortName  string
	RemotePortName string
	RemoteSysName  string
}

func main() {
	var ip = flag.String("ip", "127.0.0.1", "IP address of target device")
	var community = flag.String("c", "public", "SNMP community")
	var format = flag.String("o", "csv", "Output format (csv, json)")
	var localPortType = flag.String("lt", "desc", "port-id-subtype selection for local (desc, id)")
	var remotePortType = flag.String("rt", "desc", "port-id-subtype selection for remote (desc, id)")
	var prune = flag.Bool("p", false, "do not output LLDP entry which has no remote info")

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

	var results []gosnmp.SnmpPDU

	if *localPortType == "desc" {
		results, err = target.BulkWalkAll(oids["lldpLocPortDesc"])
	} else if *localPortType == "id" {
		results, err = target.BulkWalkAll(oids["lldpLocPortId"])
	}
	if err != nil {
		log.Fatal(err)
	}

	for _, pdu := range results {
		ifDescr := string(pdu.Value.([]uint8))
		lldpEntries[pdu.Name[26:]] = &lldpEntry{LocalPortName: ifDescr}
	}

	results, err = target.BulkWalkAll(oids["lldpRemSysName"])
	if err != nil {
		log.Fatal(err)
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
	if err != nil {
		log.Fatal(err)
	}

	for _, pdu := range results {
		remotePort := string(pdu.Value.([]uint8))
		lldpEntries[strings.Split(pdu.Name[26:], ".")[1]].RemotePortName = remotePort
	}

	if *prune {
		for key, lldp := range lldpEntries {
			if lldp.RemotePortName == "" || lldp.RemoteSysName == "" {
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
		os.Exit(0)
	} else if *format == "csv" {
		fmt.Println("Local,RemotePort,RemoteSysName")
		for _, lldp := range lldpEntries {
			fmt.Println(
				lldp.LocalPortName, ",",
				lldp.RemotePortName, ",",
				lldp.RemoteSysName,
			)
		}
		os.Exit(0)
	}

}
