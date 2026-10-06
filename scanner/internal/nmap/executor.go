package nmap

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net"
	"os/exec"
	"strings"
)

// Executor kapselt die Ausführung von Nmap.
type Executor struct{}

// NewExecutor erstellt eine neue Nmap-Ausführungskomponente.
func NewExecutor() *Executor {
	return &Executor{}
}

// ValidateTarget prüft, ob das angegebene Ziel für einen Scan zugelassen ist.
func (e *Executor) ValidateTarget(target string) error {
	target = strings.TrimSpace(target)

	if target == "" {
		return errors.New("scan target darf nicht leer sein")
	}

	if ip := net.ParseIP(target); ip != nil {
		if ip.To4() == nil {
			return errors.New("aktuell werden nur IPv4-Adressen unterstützt")
		}

		if !ip.IsPrivate() && !ip.IsLoopback() {
			return errors.New("nur private oder lokale IP-Adressen sind erlaubt")
		}

		return nil
	}

	ip, ipNet, err := net.ParseCIDR(target)
	if err != nil {
		return errors.New("ungültige IP-Adresse oder CIDR-Angabe")
	}

	if ip.To4() == nil {
		return errors.New("aktuell werden nur IPv4-Netze unterstützt")
	}

	if !ip.IsPrivate() && !ip.IsLoopback() {
		return errors.New("nur private oder lokale Netzwerke sind erlaubt")
	}

	ones, bits := ipNet.Mask.Size()
	if bits != 32 {
		return errors.New("ungültige IPv4-Netzmaske")
	}

	if ones < 24 {
		return errors.New("netzwerke grösser als /24 sind nicht erlaubt")
	}

	return nil
}

// Run führt einen kontrollierten TCP-Connect-Scan mit
// zusätzlicher Diensterkennung aus.
func (e *Executor) Run(ctx context.Context, target string) ([]byte, error) {
	if err := e.ValidateTarget(target); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(
		ctx,
		"nmap",
		"-sT",
		"-sV",
		"-oX",
		"-",
		target,
	)

	// stdout enthält das XML-Ergebnis. stderr wird getrennt
	// gesammelt, damit technische Details geloggt werden können.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {

		// Ein abgelaufenes Zeitlimit wird ausdrücklich
		// von einem allgemeinen Nmap-Fehler unterschieden.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errors.New(
				"nmap-scan wurde wegen überschreitung des zeitlimits abgebrochen",
			)
		}

		// Technische Nmap-Details werden nur im Container-Log
		// protokolliert und nicht direkt dem Benutzer angezeigt.
		if message := strings.TrimSpace(stderr.String()); message != "" {
			log.Printf("Nmap-Fehler: %s", message)
		} else {
			log.Printf("Nmap-Ausführung fehlgeschlagen: %v", err)
		}

		return nil, errors.New(
			"nmap-scan konnte nicht ausgeführt werden",
		)
	}

	return output, nil
}
