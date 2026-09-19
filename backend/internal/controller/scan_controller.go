package controller

import (
	"encoding/json"
	"net/http"

	"network-scanner/backend/internal/model"
	"network-scanner/backend/internal/service"
)

// ScanController verarbeitet die Scan-Anfragen des Clients.
// Die eigentliche Geschäftslogik wird über den ScanService ausgeführt.
type ScanController struct {
	scanService *service.ScanService
}

// NewScanController erstellt einen neuen Controller
// für das Starten und Abfragen von Scanaufträgen.
func NewScanController(scanService *service.ScanService) *ScanController {
	return &ScanController{
		scanService: scanService,
	}
}

// CreateScan nimmt ein Scan-Ziel entgegen und
// übergibt den Auftrag an den ScanService.
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

	job, err := c.scanService.CreateScan(r.Context(), request.Target)
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

	job, err := c.scanService.GetScan(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "scanauftrag nicht gefunden",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(job)
}

// GetResults liefert die strukturierten Scanergebnisse als JSON.
// Das Frontend muss dadurch keine Nmap-XML-Daten selbst verarbeiten.
func (c *ScanController) GetResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.PathValue("id")

	result, err := c.scanService.GetResults(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "scanergebnisse konnten nicht geladen werden",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(result)
}

// GetHistory stellt die gespeicherten Scanläufe als JSON bereit.
// Die Historie wird später in der Weboberfläche angezeigt.
func (c *ScanController) GetHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	history, err := c.scanService.GetHistory(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "scan-historie konnte nicht geladen werden",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(history)
}
