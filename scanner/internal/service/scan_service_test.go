package service

import (
	"errors"
	"testing"

	"network-scanner/scanner/internal/model"
	"network-scanner/scanner/internal/nmap"
)

func TestCreateScanQueueFull(t *testing.T) {
	scanService := &ScanService{
		executor: nmap.NewExecutor(),
		jobs:     make(map[string]*model.ScanJob),
		queue:    make(chan string, maxQueuedScans),
	}

	// Warteschlange vollständig belegen.
	for i := 0; i < maxQueuedScans; i++ {
		scanService.queue <- "test"
	}

	_, err := scanService.CreateScan("127.0.0.1")

	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf(
			"erwartet wurde ErrQueueFull, erhalten: %v",
			err,
		)
	}
}

func TestGetScanNotFound(t *testing.T) {
	scanService := &ScanService{
		executor: nmap.NewExecutor(),
		jobs:     make(map[string]*model.ScanJob),
		queue:    make(chan string, maxQueuedScans),
	}

	_, err := scanService.GetScan("unbekannte-id")

	if !errors.Is(err, ErrScanNotFound) {
		t.Fatalf(
			"erwartet wurde ErrScanNotFound, erhalten: %v",
			err,
		)
	}
}
