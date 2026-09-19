package nmapxml

import (
	"testing"
)

// TestParse prüft, ob die relevanten Informationen
// eines Nmap-Scans korrekt aus der XML-Ausgabe übernommen werden.
func TestParse(t *testing.T) {
	xmlData := []byte(`
		<nmaprun>
			<host>
				<status state="up"/>

				<address addr="192.168.1.10" addrtype="ipv4"/>

				<hostnames>
					<hostname name="test-server"/>
				</hostnames>

				<ports>
					<port protocol="tcp" portid="80">
						<state state="open"/>
						<service
							name="http"
							product="Apache"
							version="2.4"
						/>
					</port>

					<port protocol="tcp" portid="443">
						<state state="closed"/>
					</port>
				</ports>
			</host>
		</nmaprun>
	`)

	result, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("XML-Verarbeitung fehlgeschlagen: %v", err)
	}

	// Prüfen, ob genau ein Host erkannt wurde.
	if len(result.Hosts) != 1 {
		t.Fatalf("erwartet: 1 Host, erhalten: %d", len(result.Hosts))
	}

	host := result.Hosts[0]

	if host.IPAddress != "192.168.1.10" {
		t.Errorf("falsche IP-Adresse: %s", host.IPAddress)
	}

	if host.Hostname != "test-server" {
		t.Errorf("falscher Hostname: %s", host.Hostname)
	}

	if host.State != "up" {
		t.Errorf("falscher Hoststatus: %s", host.State)
	}

	// Prüfen, ob beide Ports dem Host zugeordnet wurden.
	if len(host.Ports) != 2 {
		t.Fatalf("erwartet: 2 Ports, erhalten: %d", len(host.Ports))
	}

	port := host.Ports[0]

	if port.Number != 80 {
		t.Errorf("falsche Portnummer: %d", port.Number)
	}

	if port.Protocol != "tcp" {
		t.Errorf("falsches Protokoll: %s", port.Protocol)
	}

	if port.State != "open" {
		t.Errorf("falscher Portstatus: %s", port.State)
	}

	// Dienstinformationen werden nur geprüft,
	// wenn Nmap tatsächlich einen Dienst erkannt hat.
	if port.Service == nil {
		t.Fatal("Dienstinformationen fehlen")
	}

	if port.Service.Name != "http" {
		t.Errorf("falscher Dienst: %s", port.Service.Name)
	}

	if port.Service.Product != "Apache" {
		t.Errorf("falsches Produkt: %s", port.Service.Product)
	}

	if port.Service.Version != "2.4" {
		t.Errorf("falsche Version: %s", port.Service.Version)
	}

	// Beim zweiten Port wurden keine Dienstinformationen geliefert.
	if host.Ports[1].Service != nil {
		t.Error("unerwartete Dienstinformationen beim zweiten Port")
	}
}

// TestParseInvalidXML prüft, ob fehlerhafte XML-Daten
// kontrolliert mit einer Fehlermeldung abgelehnt werden.
func TestParseInvalidXML(t *testing.T) {
	xmlData := []byte(`<nmaprun><host>`)

	_, err := Parse(xmlData)

	if err == nil {
		t.Error("erwarteter Fehler bei ungültigem XML ist ausgeblieben")
	}
}
