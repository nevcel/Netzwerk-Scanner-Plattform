package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"network-scanner/scanner/internal/model"
	"network-scanner/scanner/internal/service"
)

// ScanController verarbeitet die HTTP-Anfragen für Scanaufträge.
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
func (c *ScanController) CreateScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request model.ScanRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"ungültige JSON-Anfrage",
		)
		return
	}

	job, err := c.scanService.CreateScan(request.Target)
	if err != nil {

		// Eine volle Warteschlange ist kein Eingabefehler.
		// Der Scanner ist momentan lediglich ausgelastet.
		if errors.Is(err, service.ErrQueueFull) {
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"scan-warteschlange ist voll",
			)
			return
		}

		// Validierungsfehler werden als fehlerhafte Anfrage behandelt.
		writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(job)
}

// GetScan liefert den aktuellen Status eines bestehenden Scanauftrags.
func (c *ScanController) GetScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.PathValue("id")

	job, err := c.scanService.GetScan(id)
	if err != nil {
		writeJSONError(
			w,
			http.StatusNotFound,
			"scanauftrag nicht gefunden",
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(job)
}

// writeJSONError stellt sicher, dass auch Fehlerantworten
// einheitlich als JSON zurückgegeben werden.
func writeJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
