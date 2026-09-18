package controller

import (
	"encoding/json"
	"net/http"

	"network-scanner/scanner/internal/model"
	"network-scanner/scanner/internal/service"
)

// ScanController verarbeitet die HTTP-Anfragen für Scanaufträge.
// Die eigentliche Scanlogik bleibt bewusst im ScanService.
type ScanController struct {
	scanService *service.ScanService
}

// NewScanController erstellt einen Controller mit dem benötigten ScanService.
func NewScanController(scanService *service.ScanService) *ScanController {
	return &ScanController{
		scanService: scanService,
	}
}

// CreateScan nimmt einen neuen Scanauftrag entgegen.
// Bei gültiger Eingabe wird der Auftrag asynchron gestartet.
func (c *ScanController) CreateScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request model.ScanRequest

	// Der Request-Body wird ausschliesslich als JSON akzeptiert.
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error":"ungültige JSON-Anfrage"}`, http.StatusBadRequest)
		return
	}

	job, err := c.scanService.CreateScan(request.Target)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// HTTP 202 zeigt, dass der Auftrag angenommen wurde,
	// die Verarbeitung jedoch noch im Hintergrund läuft.
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(job)
}

// GetScan liefert den aktuellen Status eines bestehenden Scanauftrags.
func (c *ScanController) GetScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.PathValue("id")

	job, err := c.scanService.GetScan(id)
	if err != nil {
		http.Error(w, `{"error":"scanauftrag nicht gefunden"}`, http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(job)
}
