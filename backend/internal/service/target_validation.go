package service

import (
	"errors"
	"net"
	"strings"
)

// ValidateTarget prüft das Scan-Ziel bereits im Backend.
// Die Scanner-Komponente führt dieselbe Prüfung nochmals durch.
// Dadurch wird die Eingabe auf beiden Systemgrenzen kontrolliert.
func ValidateTarget(target string) error {
	target = strings.TrimSpace(target)

	if target == "" {
		return errors.New("scan target darf nicht leer sein")
	}

	// Einzelne IPv4-Adresse prüfen.
	if ip := net.ParseIP(target); ip != nil {
		if ip.To4() == nil {
			return errors.New("aktuell werden nur IPv4-Adressen unterstützt")
		}

		if !ip.IsPrivate() && !ip.IsLoopback() {
			return errors.New("nur private oder lokale IP-Adressen sind erlaubt")
		}

		return nil
	}

	// CIDR-Netz prüfen.
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

	// Um unbeabsichtigt grosse Scans zu verhindern,
	// werden maximal /24-Netze zugelassen.
	if ones < 24 {
		return errors.New("netzwerke grösser als /24 sind nicht erlaubt")
	}

	return nil
}
