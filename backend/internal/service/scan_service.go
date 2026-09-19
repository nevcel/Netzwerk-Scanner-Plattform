package service

import (
	"context"
	"fmt"

	"network-scanner/backend/internal/model"
	"network-scanner/backend/internal/nmapxml"
	"network-scanner/backend/internal/repository"
)

// ScanService koordiniert die Kommunikation mit dem Scanner
// und die dauerhafte Speicherung der Scanaufträge.
type ScanService struct {
	scanner    *ScannerService
	repository *repository.ScanRepository
}

// NewScanService erstellt den zentralen Service für Scanaufträge.
func NewScanService(
	scanner *ScannerService,
	repository *repository.ScanRepository,
) *ScanService {

	return &ScanService{
		scanner:    scanner,
		repository: repository,
	}
}

// CreateScan startet einen neuen Scanauftrag und speichert
// die erhaltene Scan-ID zusammen mit dem aktuellen Status.
func (s *ScanService) CreateScan(
	ctx context.Context,
	target string,
) (*model.ScanJob, error) {

	job, err := s.scanner.CreateScan(ctx, target)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, job); err != nil {
		return nil, fmt.Errorf(
			"scan wurde gestartet, konnte aber nicht gespeichert werden: %w",
			err,
		)
	}

	return job, nil
}

// GetScan fragt den aktuellen Scanstatus ab und aktualisiert
// die gespeicherten Informationen in PostgreSQL.
func (s *ScanService) GetScan(
	ctx context.Context,
	id string,
) (*model.ScanJob, error) {

	job, err := s.scanner.GetScan(ctx, id)

	if err != nil {
		// Ist der Scanner nicht erreichbar, wird versucht,
		// den zuletzt gespeicherten Zustand zurückzugeben.
		return s.repository.GetByScanID(ctx, id)
	}

	var details *nmapxml.ScanResult

	// Erst nach erfolgreichem Abschluss wird das XML verarbeitet.
	// Laufende oder fehlgeschlagene Scans besitzen noch kein
	// vollständiges Ergebnis, das ausgewertet werden könnte.
	if job.Status == "completed" {

		details, err = nmapxml.Parse([]byte(job.Output))
		if err != nil {
			return nil, fmt.Errorf(
				"scanergebnis konnte nicht ausgewertet werden: %w",
				err,
			)
		}
	}

	// Status und gegebenenfalls strukturierte Ergebnisse
	// werden gemeinsam über das Repository gespeichert.
	if err := s.repository.Update(ctx, job, details); err != nil {
		return nil, fmt.Errorf(
			"scanergebnis konnte nicht gespeichert werden: %w",
			err,
		)
	}

	return job, nil
}
