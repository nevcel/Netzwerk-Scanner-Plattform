package model

import "time"

// ScanRequest enthält die vom Backend übergebenen Angaben
// für einen neuen Netzwerkscan.
type ScanRequest struct {
	Target string `json:"target"`
}

// ScanStatus beschreibt den aktuellen Zustand eines Scanauftrags.
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
)

// ScanJob bildet einen einzelnen Scanauftrag ab.
// Die Daten werden vorerst im Arbeitsspeicher gehalten.
// Die dauerhafte Speicherung erfolgt später über PostgreSQL.
type ScanJob struct {
	ID        string     `json:"id"`
	Target    string     `json:"target"`
	Status    ScanStatus `json:"status"`
	Output    string     `json:"output,omitempty"`
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
