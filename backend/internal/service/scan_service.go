package service

import (
	"context"
	"fmt"

	"network-scanner/backend/internal/model"
	"network-scanner/backend/internal/repository"
)

// ScanService verbindet die Scanner-Komponente mit der Persistenz.
// Dadurch bleibt der Controller frei von Datenbank- und Scannerlogik.
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

// CreateScan startet einen Scan über die Scanner-Komponente
// und speichert den erhaltenen Auftrag anschliessend in PostgreSQL.
func (s *ScanService) CreateScan(
	ctx context.Context,
	target string,
) (*model.ScanJob, error) {

	job, err := s.scanner.CreateScan(ctx, target)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("scan wurde gestartet, konnte aber nicht gespeichert werden: %w", err)
	}

	return job, nil
}

// GetScan fragt den aktuellen Zustand beim Scanner ab
// und aktualisiert den gespeicherten Datensatz.
func (s *ScanService) GetScan(
	ctx context.Context,
	id string,
) (*model.ScanJob, error) {

	job, err := s.scanner.GetScan(ctx, id)

	if err != nil {
		// Ist die Scanner-Komponente nicht erreichbar, wird versucht,
		// den zuletzt gespeicherten Zustand aus PostgreSQL zu liefern.
		return s.repository.GetByScanID(ctx, id)
	}

	if err := s.repository.Update(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}
