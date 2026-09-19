package nmapxml

import (
	"encoding/xml"
	"fmt"
)

// ScanResult enthält die strukturierten Ergebnisse eines Nmap-Scans.
// Jeder Scan kann mehrere erkannte Hosts enthalten.
type ScanResult struct {
	Hosts []Host
}

// Host enthält die Informationen eines erkannten Netzwerkgeräts.
type Host struct {
	IPAddress string
	Hostname  string
	State     string
	Ports     []Port
}

// Port enthält die Informationen eines erkannten Netzwerkports.
type Port struct {
	Number   int
	Protocol string
	State    string
	Service  *Service
}

// Service enthält die von Nmap erkannten Dienstinformationen.
type Service struct {
	Name    string
	Product string
	Version string
}

// Die folgenden Strukturen bilden die relevanten Elemente
// der XML-Ausgabe von Nmap ab.
type nmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status struct {
		State string `xml:"state,attr"`
	} `xml:"status"`

	Addresses []struct {
		Address string `xml:"addr,attr"`
		Type    string `xml:"addrtype,attr"`
	} `xml:"address"`

	Hostnames []struct {
		Name string `xml:"name,attr"`
	} `xml:"hostnames>hostname"`

	Ports []nmapPort `xml:"ports>port"`
}

type nmapPort struct {
	Number   int    `xml:"portid,attr"`
	Protocol string `xml:"protocol,attr"`

	State struct {
		State string `xml:"state,attr"`
	} `xml:"state"`

	Service *struct {
		Name    string `xml:"name,attr"`
		Product string `xml:"product,attr"`
		Version string `xml:"version,attr"`
	} `xml:"service"`
}

// Parse verarbeitet die XML-Ausgabe von Nmap und wandelt
// die relevanten Informationen in eigene Go-Datenstrukturen um.
func Parse(data []byte) (*ScanResult, error) {
	var document nmapRun

	if err := xml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("nmap-xml konnte nicht verarbeitet werden: %w", err)
	}

	result := &ScanResult{
		Hosts: make([]Host, 0, len(document.Hosts)),
	}

	// Jeder erkannte Host wird einzeln ausgewertet.
	for _, xmlHost := range document.Hosts {
		host := Host{
			State: xmlHost.Status.State,
			Ports: make([]Port, 0, len(xmlHost.Ports)),
		}

		// Aktuell werden ausschliesslich IPv4-Adressen verarbeitet.
		for _, address := range xmlHost.Addresses {
			if address.Type == "ipv4" {
				host.IPAddress = address.Address
				break
			}
		}

		if host.IPAddress == "" {
			continue
		}

		// Falls Nmap einen Hostnamen erkannt hat,
		// wird der erste vorhandene Name übernommen.
		if len(xmlHost.Hostnames) > 0 {
			host.Hostname = xmlHost.Hostnames[0].Name
		}

		// Die einzelnen Ports werden dem jeweiligen Host zugeordnet.
		for _, xmlPort := range xmlHost.Ports {
			port := Port{
				Number:   xmlPort.Number,
				Protocol: xmlPort.Protocol,
				State:    xmlPort.State.State,
			}

			// Dienstinformationen sind optional und werden nur
			// übernommen, wenn Nmap entsprechende Angaben liefert.
			if xmlPort.Service != nil {
				port.Service = &Service{
					Name:    xmlPort.Service.Name,
					Product: xmlPort.Service.Product,
					Version: xmlPort.Service.Version,
				}
			}

			host.Ports = append(host.Ports, port)
		}

		result.Hosts = append(result.Hosts, host)
	}

	return result, nil
}
