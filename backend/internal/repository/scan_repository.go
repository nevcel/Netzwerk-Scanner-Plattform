package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"network-scanner/backend/internal/model"
	"network-scanner/backend/internal/nmapxml"
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

// Update aktualisiert einen Scanauftrag und speichert bei einem
// abgeschlossenen Scan zusätzlich die erkannten Hosts, Ports und Dienste.
//
// Alle Datenbankänderungen erfolgen innerhalb einer Transaktion.
// Dadurch werden unvollständige Scanergebnisse bei einem Fehler vermieden.
func (r *ScanRepository) Update(
	ctx context.Context,
	job *model.ScanJob,
	details *nmapxml.ScanResult,
) error {

	// Transaktion starten, damit sämtliche Änderungen
	// gemeinsam gespeichert oder zurückgesetzt werden.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("datenbanktransaktion konnte nicht gestartet werden: %w", err)
	}
	defer tx.Rollback(ctx)

	// Die interne Datenbank-ID des Scanauftrags ermitteln.
	var databaseID int64

	err = tx.QueryRow(
		ctx,
		`SELECT id FROM scans WHERE scan_id = $1 FOR UPDATE`,
		job.ID,
	).Scan(&databaseID)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("scanauftrag nicht in der datenbank gefunden")
	}

	if err != nil {
		return fmt.Errorf("scanauftrag konnte nicht gelesen werden: %w", err)
	}

	// Status, XML-Ausgabe und Fehlermeldung aktualisieren.
	_, err = tx.Exec(
		ctx,
		`
		UPDATE scans
		SET
			status = $1,
			raw_output = $2,
			error_message = $3,
			updated_at = $4
		WHERE id = $5
		`,
		job.Status,
		job.Output,
		job.Error,
		job.UpdatedAt,
		databaseID,
	)

	if err != nil {
		return fmt.Errorf("scanauftrag konnte nicht aktualisiert werden: %w", err)
	}

	// Strukturierte Ergebnisse werden erst gespeichert,
	// wenn Nmap den Scan erfolgreich abgeschlossen hat.
	if job.Status == "completed" {

		if details == nil {
			return errors.New("strukturierte scanergebnisse fehlen")
		}

		// Vorhandene Hostdaten dieses Scanlaufs entfernen.
		// Dadurch entstehen bei erneuten Statusabfragen keine Duplikate.
		// Zugehörige Ports und Dienste werden über ON DELETE CASCADE entfernt.
		_, err = tx.Exec(
			ctx,
			`DELETE FROM hosts WHERE scan_id = $1`,
			databaseID,
		)

		if err != nil {
			return fmt.Errorf("vorhandene hostdaten konnten nicht entfernt werden: %w", err)
		}

		// Jeden erkannten Host in PostgreSQL speichern.
		for _, host := range details.Hosts {

			var hostID int64

			err = tx.QueryRow(
				ctx,
				`
				INSERT INTO hosts (
					scan_id,
					ip_address,
					hostname,
					state
				)
				VALUES ($1, $2, $3, $4)
				RETURNING id
				`,
				databaseID,
				host.IPAddress,
				host.Hostname,
				host.State,
			).Scan(&hostID)

			if err != nil {
				return fmt.Errorf("host konnte nicht gespeichert werden: %w", err)
			}

			// Die erkannten Ports werden dem jeweiligen Host zugeordnet.
			for _, port := range host.Ports {

				var portID int64

				err = tx.QueryRow(
					ctx,
					`
					INSERT INTO ports (
						host_id,
						port_number,
						protocol,
						state
					)
					VALUES ($1, $2, $3, $4)
					RETURNING id
					`,
					hostID,
					port.Number,
					port.Protocol,
					port.State,
				).Scan(&portID)

				if err != nil {
					return fmt.Errorf("port konnte nicht gespeichert werden: %w", err)
				}

				// Dienstinformationen werden nur gespeichert,
				// wenn Nmap entsprechende Angaben geliefert hat.
				if port.Service != nil {

					_, err = tx.Exec(
						ctx,
						`
						INSERT INTO services (
							port_id,
							name,
							product,
							version
						)
						VALUES ($1, $2, $3, $4)
						`,
						portID,
						port.Service.Name,
						port.Service.Product,
						port.Service.Version,
					)

					if err != nil {
						return fmt.Errorf("dienst konnte nicht gespeichert werden: %w", err)
					}
				}
			}
		}
	}

	// Erst wenn sämtliche Datenbankoperationen erfolgreich waren,
	// werden die Änderungen dauerhaft übernommen.
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("scanergebnis konnte nicht gespeichert werden: %w", err)
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
