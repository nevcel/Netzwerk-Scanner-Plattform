package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"network-scanner/backend/internal/model"
)

// ScanRepository kapselt sämtliche Datenbankzugriffe,
// welche die Tabelle scans betreffen.
type ScanRepository struct {
	db *pgxpool.Pool
}

// NewScanRepository erstellt ein neues Repository
// mit dem bestehenden PostgreSQL-Connection-Pool.
func NewScanRepository(db *pgxpool.Pool) *ScanRepository {
	return &ScanRepository{
		db: db,
	}
}

// Create speichert einen neu erstellten Scanauftrag in PostgreSQL.
func (r *ScanRepository) Create(
	ctx context.Context,
	job *model.ScanJob,
) error {

	query := `
		INSERT INTO scans (
			scan_id,
			target,
			status,
			raw_output,
			error_message,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		job.ID,
		job.Target,
		job.Status,
		job.Output,
		job.Error,
		job.CreatedAt,
		job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("scan konnte nicht gespeichert werden: %w", err)
	}

	return nil
}

// Update aktualisiert Status, Resultat und Fehlermeldung
// eines bereits gespeicherten Scanauftrags.
func (r *ScanRepository) Update(
	ctx context.Context,
	job *model.ScanJob,
) error {

	query := `
		UPDATE scans
		SET
			status = $1,
			raw_output = $2,
			error_message = $3,
			updated_at = $4
		WHERE scan_id = $5
	`

	result, err := r.db.Exec(
		ctx,
		query,
		job.Status,
		job.Output,
		job.Error,
		job.UpdatedAt,
		job.ID,
	)

	if err != nil {
		return fmt.Errorf("scan konnte nicht aktualisiert werden: %w", err)
	}

	// Damit werden versehentliche Updates auf nicht vorhandene
	// Scanaufträge erkannt.
	if result.RowsAffected() == 0 {
		return errors.New("scanauftrag nicht in der datenbank gefunden")
	}

	return nil
}

// GetByScanID liest einen Scanauftrag anhand seiner externen Scan-ID.
func (r *ScanRepository) GetByScanID(
	ctx context.Context,
	scanID string,
) (*model.ScanJob, error) {

	query := `
		SELECT
			scan_id,
			target,
			status,
			COALESCE(raw_output, ''),
			COALESCE(error_message, ''),
			created_at,
			updated_at
		FROM scans
		WHERE scan_id = $1
	`

	var job model.ScanJob

	err := r.db.QueryRow(
		ctx,
		query,
		scanID,
	).Scan(
		&job.ID,
		&job.Target,
		&job.Status,
		&job.Output,
		&job.Error,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("scanauftrag nicht gefunden")
	}

	if err != nil {
		return nil, fmt.Errorf("scan konnte nicht gelesen werden: %w", err)
	}

	return &job, nil
}
