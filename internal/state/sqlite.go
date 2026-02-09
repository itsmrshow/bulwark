package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/itsmrshow/bulwark/internal/logging"
)

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	db     *sql.DB
	logger *logging.Logger
	path   string
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(path string, logger *logging.Logger) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	return &SQLiteStore{
		db:     db,
		logger: logger.WithComponent("sqlite-store"),
		path:   path,
	}, nil
}

// Initialize creates tables and runs migrations
func (s *SQLiteStore) Initialize(ctx context.Context) error {
	s.logger.Info().Str("path", s.path).Msg("Initializing SQLite database")

	schema := `
		-- Targets table
		CREATE TABLE IF NOT EXISTS targets (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			labels_json TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(name)
		);

		-- Services table
		CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			target_id TEXT NOT NULL,
			name TEXT NOT NULL,
			image TEXT NOT NULL,
			current_digest TEXT,
			labels_json TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY(target_id) REFERENCES targets(id) ON DELETE CASCADE,
			UNIQUE(target_id, name)
		);

		-- Update history table
		CREATE TABLE IF NOT EXISTS update_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id TEXT NOT NULL,
			service_id TEXT NOT NULL,
			service_name TEXT NOT NULL,
			old_digest TEXT NOT NULL,
			new_digest TEXT NOT NULL,
			success BOOLEAN NOT NULL,
			error TEXT,
			probe_results_json TEXT,
			rollback_performed BOOLEAN NOT NULL DEFAULT 0,
			rollback_digest TEXT,
			started_at DATETIME NOT NULL,
			completed_at DATETIME NOT NULL,
			FOREIGN KEY(target_id) REFERENCES targets(id) ON DELETE CASCADE,
			FOREIGN KEY(service_id) REFERENCES services(id) ON DELETE CASCADE
		);

		-- Indices for common queries
		CREATE INDEX IF NOT EXISTS idx_targets_name ON targets(name);
		CREATE INDEX IF NOT EXISTS idx_targets_updated_at ON targets(updated_at);
		CREATE INDEX IF NOT EXISTS idx_services_target_id ON services(target_id);
		CREATE INDEX IF NOT EXISTS idx_services_image ON services(image);
		CREATE INDEX IF NOT EXISTS idx_update_history_target_id ON update_history(target_id);
		CREATE INDEX IF NOT EXISTS idx_update_history_service_id ON update_history(service_id);
		CREATE INDEX IF NOT EXISTS idx_update_history_completed_at ON update_history(completed_at);

		-- Settings table
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	s.logger.Info().Msg("Database schema initialized")
	return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	s.logger.Info().Msg("Closing database connection")
	return s.db.Close()
}

// SaveTarget saves or updates a target
func (s *SQLiteStore) SaveTarget(ctx context.Context, target *Target) error {
	labelsJSON, err := json.Marshal(target.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	now := time.Now()
	if target.CreatedAt.IsZero() {
		target.CreatedAt = now
	}
	target.UpdatedAt = now

	query := `
		INSERT INTO targets (id, type, name, path, labels_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			name = excluded.name,
			path = excluded.path,
			labels_json = excluded.labels_json,
			updated_at = excluded.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		target.ID,
		target.Type,
		target.Name,
		target.Path,
		string(labelsJSON),
		target.CreatedAt,
		target.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save target: %w", err)
	}

	s.logger.Debug().Str("target_id", target.ID).Str("name", target.Name).Msg("Saved target")
	return nil
}

// GetTarget retrieves a target by ID
func (s *SQLiteStore) GetTarget(ctx context.Context, id string) (*Target, error) {
	query := `SELECT id, type, name, path, labels_json, created_at, updated_at FROM targets WHERE id = ?`

	var target Target
	var labelsJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&target.ID,
		&target.Type,
		&target.Name,
		&target.Path,
		&labelsJSON,
		&target.CreatedAt,
		&target.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("target not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get target: %w", err)
	}

	if err := json.Unmarshal([]byte(labelsJSON), &target.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	// Load services
	services, err := s.GetServicesByTarget(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load services: %w", err)
	}
	target.Services = services

	return &target, nil
}

// GetTargetByName retrieves a target by name
func (s *SQLiteStore) GetTargetByName(ctx context.Context, name string) (*Target, error) {
	query := `SELECT id, type, name, path, labels_json, created_at, updated_at FROM targets WHERE name = ?`

	var target Target
	var labelsJSON string

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&target.ID,
		&target.Type,
		&target.Name,
		&target.Path,
		&labelsJSON,
		&target.CreatedAt,
		&target.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("target not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get target: %w", err)
	}

	if err := json.Unmarshal([]byte(labelsJSON), &target.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	// Load services
	services, err := s.GetServicesByTarget(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load services: %w", err)
	}
	target.Services = services

	return &target, nil
}

// ListTargets retrieves all targets
func (s *SQLiteStore) ListTargets(ctx context.Context) ([]Target, error) {
	query := `SELECT id, type, name, path, labels_json, created_at, updated_at FROM targets ORDER BY name`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list targets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var targets []Target
	for rows.Next() {
		var target Target
		var labelsJSON string

		if err := rows.Scan(
			&target.ID,
			&target.Type,
			&target.Name,
			&target.Path,
			&labelsJSON,
			&target.CreatedAt,
			&target.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan target: %w", err)
		}

		if err := json.Unmarshal([]byte(labelsJSON), &target.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}

		// Load services
		services, err := s.GetServicesByTarget(ctx, target.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load services: %w", err)
		}
		target.Services = services

		targets = append(targets, target)
	}

	return targets, rows.Err()
}

// DeleteTarget deletes a target and its services
func (s *SQLiteStore) DeleteTarget(ctx context.Context, id string) error {
	query := `DELETE FROM targets WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}

	s.logger.Debug().Str("target_id", id).Msg("Deleted target")
	return nil
}

// SaveService saves or updates a service
func (s *SQLiteStore) SaveService(ctx context.Context, service *Service) error {
	labelsJSON, err := json.Marshal(service.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	now := time.Now()
	if service.CreatedAt.IsZero() {
		service.CreatedAt = now
	}
	service.UpdatedAt = now

	query := `
		INSERT INTO services (id, target_id, name, image, current_digest, labels_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			target_id = excluded.target_id,
			name = excluded.name,
			image = excluded.image,
			current_digest = excluded.current_digest,
			labels_json = excluded.labels_json,
			updated_at = excluded.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		service.ID,
		service.TargetID,
		service.Name,
		service.Image,
		service.CurrentDigest,
		string(labelsJSON),
		service.CreatedAt,
		service.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save service: %w", err)
	}

	s.logger.Debug().Str("service_id", service.ID).Str("name", service.Name).Msg("Saved service")
	return nil
}

// GetService retrieves a service by ID
func (s *SQLiteStore) GetService(ctx context.Context, id string) (*Service, error) {
	query := `SELECT id, target_id, name, image, current_digest, labels_json, created_at, updated_at FROM services WHERE id = ?`

	var service Service
	var labelsJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&service.ID,
		&service.TargetID,
		&service.Name,
		&service.Image,
		&service.CurrentDigest,
		&labelsJSON,
		&service.CreatedAt,
		&service.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("service not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	if err := json.Unmarshal([]byte(labelsJSON), &service.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	return &service, nil
}

// GetServicesByTarget retrieves all services for a target
func (s *SQLiteStore) GetServicesByTarget(ctx context.Context, targetID string) ([]Service, error) {
	query := `SELECT id, target_id, name, image, current_digest, labels_json, created_at, updated_at FROM services WHERE target_id = ? ORDER BY name`

	rows, err := s.db.QueryContext(ctx, query, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var services []Service
	for rows.Next() {
		var service Service
		var labelsJSON string

		if err := rows.Scan(
			&service.ID,
			&service.TargetID,
			&service.Name,
			&service.Image,
			&service.CurrentDigest,
			&labelsJSON,
			&service.CreatedAt,
			&service.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}

		if err := json.Unmarshal([]byte(labelsJSON), &service.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}

		services = append(services, service)
	}

	return services, rows.Err()
}

// DeleteService deletes a service
func (s *SQLiteStore) DeleteService(ctx context.Context, id string) error {
	query := `DELETE FROM services WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	s.logger.Debug().Str("service_id", id).Msg("Deleted service")
	return nil
}

// SaveUpdateResult saves an update result to history
func (s *SQLiteStore) SaveUpdateResult(ctx context.Context, result *UpdateResult) error {
	probeResultsJSON, err := json.Marshal(result.ProbeResults)
	if err != nil {
		return fmt.Errorf("failed to marshal probe results: %w", err)
	}

	query := `
		INSERT INTO update_history (
			target_id, service_id, service_name, old_digest, new_digest,
			success, error, probe_results_json, rollback_performed, rollback_digest,
			started_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	errorStr := ""
	if result.Error != nil {
		errorStr = result.Error.Error()
	}

	_, err = s.db.ExecContext(ctx, query,
		result.TargetID,
		result.ServiceID,
		result.ServiceName,
		result.OldDigest,
		result.NewDigest,
		result.Success,
		errorStr,
		string(probeResultsJSON),
		result.RollbackPerformed,
		result.RollbackDigest,
		result.StartedAt,
		result.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save update result: %w", err)
	}

	s.logger.Debug().
		Str("service_name", result.ServiceName).
		Bool("success", result.Success).
		Msg("Saved update result")

	return nil
}

// GetUpdateHistory retrieves recent update history
func (s *SQLiteStore) GetUpdateHistory(ctx context.Context, limit int) ([]UpdateResult, error) {
	query := `
		SELECT id, target_id, service_id, service_name, old_digest, new_digest,
			   success, error, probe_results_json, rollback_performed, rollback_digest,
			   started_at, completed_at
		FROM update_history
		ORDER BY completed_at DESC
		LIMIT ?
	`

	return s.queryUpdateHistory(ctx, query, limit)
}

// GetUpdateHistoryByTarget retrieves update history for a target
func (s *SQLiteStore) GetUpdateHistoryByTarget(ctx context.Context, targetID string, limit int) ([]UpdateResult, error) {
	query := `
		SELECT id, target_id, service_id, service_name, old_digest, new_digest,
			   success, error, probe_results_json, rollback_performed, rollback_digest,
			   started_at, completed_at
		FROM update_history
		WHERE target_id = ?
		ORDER BY completed_at DESC
		LIMIT ?
	`

	return s.queryUpdateHistory(ctx, query, targetID, limit)
}

// GetUpdateHistoryByService retrieves update history for a service
func (s *SQLiteStore) GetUpdateHistoryByService(ctx context.Context, serviceID string, limit int) ([]UpdateResult, error) {
	query := `
		SELECT id, target_id, service_id, service_name, old_digest, new_digest,
			   success, error, probe_results_json, rollback_performed, rollback_digest,
			   started_at, completed_at
		FROM update_history
		WHERE service_id = ?
		ORDER BY completed_at DESC
		LIMIT ?
	`

	return s.queryUpdateHistory(ctx, query, serviceID, limit)
}

// GetLastSuccessfulUpdate retrieves the last successful update for a service
func (s *SQLiteStore) GetLastSuccessfulUpdate(ctx context.Context, serviceID string) (*UpdateResult, error) {
	query := `
		SELECT id, target_id, service_id, service_name, old_digest, new_digest,
			   success, error, probe_results_json, rollback_performed, rollback_digest,
			   started_at, completed_at
		FROM update_history
		WHERE service_id = ? AND success = 1
		ORDER BY completed_at DESC
		LIMIT 1
	`

	results, err := s.queryUpdateHistory(ctx, query, serviceID, 1)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no successful updates found for service: %s", serviceID)
	}

	return &results[0], nil
}

// queryUpdateHistory is a helper to execute update history queries
func (s *SQLiteStore) queryUpdateHistory(ctx context.Context, query string, args ...interface{}) ([]UpdateResult, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query update history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []UpdateResult
	for rows.Next() {
		var result UpdateResult
		var errorStr sql.NullString
		var probeResultsJSON string
		var id int64

		if err := rows.Scan(
			&id,
			&result.TargetID,
			&result.ServiceID,
			&result.ServiceName,
			&result.OldDigest,
			&result.NewDigest,
			&result.Success,
			&errorStr,
			&probeResultsJSON,
			&result.RollbackPerformed,
			&result.RollbackDigest,
			&result.StartedAt,
			&result.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan update result: %w", err)
		}

		if errorStr.Valid && errorStr.String != "" {
			result.Error = fmt.Errorf("%s", errorStr.String)
		}

		if err := json.Unmarshal([]byte(probeResultsJSON), &result.ProbeResults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal probe results: %w", err)
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// PruneHistory deletes old update history
func (s *SQLiteStore) PruneHistory(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM update_history WHERE completed_at < ?`
	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to prune history: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	s.logger.Info().Int64("rows_deleted", rowsAffected).Msg("Pruned old update history")

	return nil
}

// PruneStaleTargets deletes targets not updated recently
func (s *SQLiteStore) PruneStaleTargets(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM targets WHERE updated_at < ?`
	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to prune stale targets: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	s.logger.Info().Int64("rows_deleted", rowsAffected).Msg("Pruned stale targets")

	return nil
}

// GetSetting retrieves a setting value.
func (s *SQLiteStore) GetSetting(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM settings WHERE key = ?`
	var value string
	err := s.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

// SetSetting stores a setting value.
func (s *SQLiteStore) SetSetting(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at
	`
	_, err := s.db.ExecContext(ctx, query, key, value, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}
