package nmap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// Executor kapselt die Ausführung von Nmap.
// Dadurch wird verhindert, dass Nmap-Befehle direkt aus Benutzereingaben
// zusammengesetzt und ungeprüft ausgeführt werden.
type Executor struct{}

// NewExecutor erstellt eine neue Nmap-Ausführungskomponente.
func NewExecutor() *Executor {
	return &Executor{}
}

// ValidateTarget prüft, ob das angegebene Ziel für einen Scan zugelassen ist.
// Aktuell werden nur private IPv4-Adressen, localhost sowie Netze bis maximal /24 erlaubt.
func (e *Executor) ValidateTarget(target string) error {
	target = strings.TrimSpace(target)

	if target == "" {
		return errors.New("scan target darf nicht leer sein")
	}

	// Prüfung einer einzelnen IPv4-Adresse.
	if ip := net.ParseIP(target); ip != nil {
		if ip.To4() == nil {
			return errors.New("aktuell werden nur IPv4-Adressen unterstützt")
		}

		if !ip.IsPrivate() && !ip.IsLoopback() {
			return errors.New("nur private oder lokale IP-Adressen sind erlaubt")
		}

		return nil
	}

	// Prüfung eines IPv4-Netzes im CIDR-Format.
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

	// Grössere Netze als /24 werden bewusst ausgeschlossen,
	// damit unbeabsichtigt umfangreiche Scans verhindert werden.
	if ones < 24 {
		return errors.New("netzwerke grösser als /24 sind nicht erlaubt")
	}

	return nil
}

// Run führt einen kontrollierten TCP-Connect-Scan aus.
// Der Nmap-Output wird im XML-Format zurückgegeben, damit er später
// strukturiert verarbeitet und in PostgreSQL gespeichert werden kann.
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

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nmap konnte nicht ausgeführt werden: %w", err)
	}

	return output, nil
}
