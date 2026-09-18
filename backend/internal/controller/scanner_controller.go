package controller

import (
	"encoding/json"
	"net/http"

	"network-scanner/backend/internal/service"
)

// ScannerController ist für Funktionen zuständig,
// welche direkt die Scanner-Komponente betreffen.
type ScannerController struct {
	scannerService *service.ScannerService
}

// NewScannerController erstellt einen Controller
// für die Kommunikation mit der Scanner-Komponente.
func NewScannerController(scannerService *service.ScannerService) *ScannerController {
	return &ScannerController{
		scannerService: scannerService,
	}
}

// Health prüft, ob die Scanner-Komponente
// vom Backend aus erreichbar ist.
func (c *ScannerController) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := c.scannerService.Health(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"scanner": "not reachable",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"scanner": "reachable",
	})
}
