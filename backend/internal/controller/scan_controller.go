package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"network-scanner/backend/internal/model"
	"network-scanner/backend/internal/service"
)

// ScanController verarbeitet die Scan-Anfragen des Clients.
type ScanController struct {
	scanService *service.ScanService
}

func NewScanController(scanService *service.ScanService) *ScanController {
	return &ScanController{
		scanService: scanService,
	}
}

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

	job, err := c.scanService.CreateScan(r.Context(), request.Target)
	if err != nil {

		switch {
		case errors.Is(err, service.ErrInvalidTarget):
			writeJSONError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)

		case errors.Is(err, service.ErrScannerBusy):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"scanner ist momentan ausgelastet",
			)

		case errors.Is(err, service.ErrScannerUnavailable):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"scanner-komponente ist momentan nicht erreichbar",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"scanauftrag konnte nicht erstellt werden",
			)
		}

		return
	}

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(job)
}

func (c *ScanController) GetScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.PathValue("id")

	job, err := c.scanService.GetScan(r.Context(), id)
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

func (c *ScanController) GetResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.PathValue("id")

	result, err := c.scanService.GetResults(r.Context(), id)
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"scanergebnisse konnten nicht geladen werden",
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func (c *ScanController) GetHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	history, err := c.scanService.GetHistory(r.Context())
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"scan-historie konnte nicht geladen werden",
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(history)
}

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
