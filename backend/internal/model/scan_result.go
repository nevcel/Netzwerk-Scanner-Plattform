package model

// ScanResults enthält die strukturierten Ergebnisse eines Scanlaufs.
// Die Daten werden aus PostgreSQL gelesen und über die REST-API ausgegeben.
type ScanResults struct {
	ScanID string       `json:"scan_id"`
	Hosts  []HostResult `json:"hosts"`
}

// HostResult enthält die Informationen eines erkannten Netzwerkgeräts.
type HostResult struct {
	IPAddress string       `json:"ip_address"`
	Hostname  string       `json:"hostname"`
	State     string       `json:"state"`
	Ports     []PortResult `json:"ports"`
}

// PortResult enthält die erkannten Port- und Dienstinformationen.
type PortResult struct {
	Number   int            `json:"number"`
	Protocol string         `json:"protocol"`
	State    string         `json:"state"`
	Service  *ServiceResult `json:"service,omitempty"`
}

// ServiceResult enthält zusätzliche Informationen zu einem Dienst.
// Diese Angaben sind optional, da Nmap nicht jeden Dienst identifizieren kann.
type ServiceResult struct {
	Name    string `json:"name"`
	Product string `json:"product"`
	Version string `json:"version"`
}
