package model

import "time"

// ScanHistoryItem enthält die wichtigsten Informationen
// eines gespeicherten Scanlaufs für die spätere Historienansicht.
type ScanHistoryItem struct {
	ID        string    `json:"id"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	HostCount int       `json:"host_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScanHistory bildet die Antwort der REST-Schnittstelle ab.
// Mehrere Scanläufe werden als Liste zurückgegeben.
type ScanHistory struct {
	Scans []ScanHistoryItem `json:"scans"`
}
