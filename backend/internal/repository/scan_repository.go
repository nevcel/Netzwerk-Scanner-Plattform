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

// GetResults liest die strukturierten Ergebnisse eines Scanlaufs
// aus PostgreSQL und bereitet sie für die REST-API auf.
func (r *ScanRepository) GetResults(
	ctx context.Context,
	scanID string,
) (*model.ScanResults, error) {

	// Die Tabellen werden über ihre Fremdschlüssel verbunden.
	// LEFT JOIN stellt sicher, dass auch Hosts ohne erkannte Ports
	// oder Scanläufe ohne erkannte Hosts zurückgegeben werden können.
	query := `
		SELECT
			s.scan_id,
			h.id,
			COALESCE(host(h.ip_address), ''),
			COALESCE(h.hostname, ''),
			COALESCE(h.state, ''),
			p.id,
			COALESCE(p.port_number, 0),
			COALESCE(p.protocol, ''),
			COALESCE(p.state, ''),
			sv.id,
			COALESCE(sv.name, ''),
			COALESCE(sv.product, ''),
			COALESCE(sv.version, '')
		FROM scans s
		LEFT JOIN hosts h ON h.scan_id = s.id
		LEFT JOIN ports p ON p.host_id = h.id
		LEFT JOIN services sv ON sv.port_id = p.id
		WHERE s.scan_id = $1
		ORDER BY h.ip_address, p.port_number
	`

	rows, err := r.db.Query(ctx, query, scanID)
	if err != nil {
		return nil, fmt.Errorf(
			"scanergebnisse konnten nicht gelesen werden: %w",
			err,
		)
	}
	defer rows.Close()

	result := &model.ScanResults{
		ScanID: scanID,
		Hosts:  make([]model.HostResult, 0),
	}

	// Die interne Host-ID dient zur Zuordnung der Ports.
	// Dadurch wird jeder Host nur einmal in die Antwort übernommen.
	hostIndex := make(map[int64]int)

	found := false

	for rows.Next() {
		found = true

		var returnedScanID string

		// IDs können bei LEFT JOIN NULL sein.
		// Deshalb werden Pointer für optionale Datensätze verwendet.
		var hostID *int64
		var portID *int64
		var serviceID *int64

		var ipAddress string
		var hostname string
		var hostState string

		var portNumber int
		var protocol string
		var portState string

		var serviceName string
		var product string
		var version string

		err := rows.Scan(
			&returnedScanID,
			&hostID,
			&ipAddress,
			&hostname,
			&hostState,
			&portID,
			&portNumber,
			&protocol,
			&portState,
			&serviceID,
			&serviceName,
			&product,
			&version,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"scanergebnis konnte nicht verarbeitet werden: %w",
				err,
			)
		}

		// Ein Scanlauf kann existieren, ohne dass Hosts erkannt wurden.
		if hostID == nil {
			continue
		}

		index, exists := hostIndex[*hostID]

		if !exists {
			result.Hosts = append(
				result.Hosts,
				model.HostResult{
					IPAddress: ipAddress,
					Hostname:  hostname,
					State:     hostState,
					Ports:     make([]model.PortResult, 0),
				},
			)

			index = len(result.Hosts) - 1
			hostIndex[*hostID] = index
		}

		// Hosts ohne erkannte Ports werden ebenfalls berücksichtigt.
		if portID == nil {
			continue
		}

		port := model.PortResult{
			Number:   portNumber,
			Protocol: protocol,
			State:    portState,
		}

		// Dienstinformationen sind optional.
		if serviceID != nil {
			port.Service = &model.ServiceResult{
				Name:    serviceName,
				Product: product,
				Version: version,
			}
		}

		result.Hosts[index].Ports = append(
			result.Hosts[index].Ports,
			port,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"fehler beim Lesen der Scanergebnisse: %w",
			err,
		)
	}

	// Falls keine Zeile zurückgegeben wurde,
	// existiert die angeforderte Scan-ID nicht.
	if !found {
		return nil, errors.New("scanauftrag nicht gefunden")
	}

	return result, nil
}

// GetHistory liest die letzten 50 Scanläufe aus PostgreSQL.
// Die Ergebnisse werden nach Erstellungsdatum absteigend sortiert,
// sodass der neueste Scan zuerst angezeigt wird.
func (r *ScanRepository) GetHistory(
	ctx context.Context,
) (*model.ScanHistory, error) {

	query := `
		SELECT
			s.scan_id,
			s.target,
			s.status,
			COUNT(h.id) AS host_count,
			s.created_at,
			s.updated_at
		FROM scans s
		LEFT JOIN hosts h ON h.scan_id = s.id
		GROUP BY s.id
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT 50
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf(
			"scan-historie konnte nicht geladen werden: %w",
			err,
		)
	}
	defer rows.Close()

	history := &model.ScanHistory{
		Scans: make([]model.ScanHistoryItem, 0),
	}

	// Jeder Scanlauf wird in ein eigenes Datenmodell übertragen.
	for rows.Next() {
		var item model.ScanHistoryItem

		if err := rows.Scan(
			&item.ID,
			&item.Target,
			&item.Status,
			&item.HostCount,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"scan-historie konnte nicht verarbeitet werden: %w",
				err,
			)
		}

		history.Scans = append(history.Scans, item)
	}

	// Auch Fehler, die während des Lesens der Datensätze
	// auftreten, werden kontrolliert zurückgegeben.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"fehler beim Lesen der Scan-Historie: %w",
			err,
		)
	}

	return history, nil
}
