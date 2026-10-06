package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"network-scanner/backend/internal/service"
)

func TestCreateScanInvalidJSON(t *testing.T) {
	scannerService := service.NewScannerService("http://127.0.0.1:1")
	scanService := service.NewScanService(scannerService, nil)
	controller := NewScanController(scanService)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/scans",
		strings.NewReader(`{"target":`),
	)

	response := httptest.NewRecorder()

	controller.CreateScan(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"erwartet wurde HTTP 400, erhalten: %d",
			response.Code,
		)
	}
}

func TestCreateScanPublicTargetRejected(t *testing.T) {
	scannerService := service.NewScannerService("http://127.0.0.1:1")
	scanService := service.NewScanService(scannerService, nil)
	controller := NewScanController(scanService)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/scans",
		strings.NewReader(`{"target":"8.8.8.8"}`),
	)

	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	controller.CreateScan(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"erwartet wurde HTTP 400, erhalten: %d",
			response.Code,
		)
	}
}

func TestCreateScanScannerUnavailable(t *testing.T) {
	scannerService := service.NewScannerService("http://127.0.0.1:1")
	scanService := service.NewScanService(scannerService, nil)
	controller := NewScanController(scanService)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/scans",
		strings.NewReader(`{"target":"127.0.0.1"}`),
	)

	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	controller.CreateScan(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"erwartet wurde HTTP 503, erhalten: %d",
			response.Code,
		)
	}
}
