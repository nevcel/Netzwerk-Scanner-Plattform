package controller

import (
	"encoding/json"
	"net/http"

	"network-scanner/backend/internal/model"
	"network-scanner/backend/internal/service"
)

// ScanController verarbeitet die Scan-Anfragen des Clients.
// Die eigentliche Kommunikation mit dem Scanner erfolgt über den ScannerService.
type ScanController struct {
	scannerService *service.ScannerService
}

// NewScanController erstellt einen neuen Controller
// für das Starten und Abfragen von Scanaufträgen.
func NewScanController(scannerService *service.ScannerService) *ScanController {
	return &ScanController{
		scannerService: scannerService,
	}
}

// CreateScan nimmt ein Scan-Ziel entgegen und
// übergibt den Auftrag an die Scanner-Komponente.
func (c *ScanController) CreateScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request model.ScanRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "ungültige JSON-Anfrage",
		})
		return
	}

	job, err := c.scannerService.CreateScan(r.Context(), request.Target)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// HTTP 202 zeigt an, dass der Scanauftrag angenommen
	// und im Hintergrund verarbeitet wird.
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(job)
}

// GetScan liefert den aktuellen Zustand
// eines bereits gestarteten Scanauftrags.
func (c *ScanController) GetScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.PathValue("id")

	job, err := c.scannerService.GetScan(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "scanauftrag nicht gefunden",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(job)
}
