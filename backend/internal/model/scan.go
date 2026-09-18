package model

import "time"

// ScanRequest enthält die vom Client übermittelten Angaben
// zum Starten eines neuen Scanauftrags.
type ScanRequest struct {
	Target string `json:"target"`
}

// ScanJob bildet den aktuellen Zustand eines Scanauftrags ab.
// Die Struktur entspricht der Antwort der Scanner-Komponente.
type ScanJob struct {
	ID        string    `json:"id"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
