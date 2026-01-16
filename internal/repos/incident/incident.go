package incident

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ocenb/geo-alerts/internal/domain/errs"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/storage/transactor"
)

type Repo struct {
	tm *transactor.Manager
}

func New(tm *transactor.Manager) *Repo {
	return &Repo{tm}
}

func (r *Repo) GetByID(ctx context.Context, id int64) (*models.Incident, error) {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		SELECT 
			id, 
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude,
			radius_meters,
			is_active, 
			created_at, 
			updated_at
		FROM incidents
		WHERE id = $1
	`

	var inc models.Incident

	err := q.QueryRow(ctx, query, id).Scan(
		&inc.ID,
		&inc.Latitude,
		&inc.Longitude,
		&inc.Radius,
		&inc.IsActive,
		&inc.CreatedAt,
		&inc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrIncidentNotFound
		}
		return nil, fmt.Errorf("failed to get incident by id: %w", err)
	}

	return &inc, nil
}

func (r *Repo) Create(ctx context.Context, params *models.CreateIncidentParams) (*models.Incident, error) {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		INSERT INTO incidents (location, radius_meters, is_active)
		SELECT 
			ST_SetSRID(ST_MakePoint($1, $2), 4326),
			$3, 
			TRUE
		WHERE NOT EXISTS (
			SELECT 1 FROM incidents 
			WHERE ST_DWithin(
				location::geography, 
				ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, 
				1.0 
			)
		)
		RETURNING 
			id, 
			ST_Y(location::geometry),
			ST_X(location::geometry),
			radius_meters, 
			is_active, 
			created_at, 
			updated_at
	`

	created := &models.Incident{}

	err := q.QueryRow(ctx, query,
		params.Longitude,
		params.Latitude,
		params.Radius,
	).Scan(
		&created.ID,
		&created.Latitude,
		&created.Longitude,
		&created.Radius,
		&created.IsActive,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrIncidentExists
		}
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}

	return created, nil
}

func (r *Repo) Update(ctx context.Context, params *models.UpdateIncidentParams) (*models.Incident, error) {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		UPDATE incidents
		SET 
			location = ST_SetSRID(ST_MakePoint($1, $2), 4326),
			radius_meters = $3,
			updated_at = NOW()
		WHERE id = $4
		  AND NOT EXISTS (
			SELECT 1 FROM incidents 
			WHERE id != $4
			AND ST_DWithin(
				location::geography, 
				ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, 
				1.0 
			)
		  )
		RETURNING 
			id, 
			ST_Y(location::geometry),
			ST_X(location::geometry),
			radius_meters, 
			is_active, 
			created_at, 
			updated_at
	`

	updated := &models.Incident{}

	err := q.QueryRow(ctx, query,
		params.Longitude,
		params.Latitude,
		params.Radius,
		params.ID,
	).Scan(
		&updated.ID,
		&updated.Latitude,
		&updated.Longitude,
		&updated.Radius,
		&updated.IsActive,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM incidents WHERE id = $1)`
			if checkErr := q.QueryRow(ctx, checkQuery, params.ID).Scan(&exists); checkErr != nil {
				return nil, fmt.Errorf("failed to check incident existence: %w", checkErr)
			}
			if !exists {
				return nil, errs.ErrIncidentNotFound
			}
			return nil, errs.ErrIncidentExists
		}
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	return updated, nil
}

func (r *Repo) Deactivate(ctx context.Context, id int64) error {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		UPDATE incidents
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
	`

	tag, err := q.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate incident: %w", err)
	}

	if tag.RowsAffected() == 0 {
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM incidents WHERE id = $1)`
		if err := q.QueryRow(ctx, checkQuery, id).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check incident existence: %w", err)
		}

		if !exists {
			return errs.ErrIncidentNotFound
		}

		return nil
	}

	return nil
}

func (r *Repo) List(ctx context.Context, limit, offset int) ([]models.Incident, error) {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		SELECT 
			id, 
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude,
			radius_meters,
			is_active, 
			created_at, 
			updated_at
		FROM incidents
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := q.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []models.Incident

	for rows.Next() {
		var inc models.Incident
		if err := rows.Scan(
			&inc.ID,
			&inc.Latitude,
			&inc.Longitude,
			&inc.Radius,
			&inc.IsActive,
			&inc.CreatedAt,
			&inc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, inc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return incidents, nil
}

func (r *Repo) GetStats(ctx context.Context, window time.Duration) ([]models.Stats, error) {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		SELECT 
			i.id,
			ST_Y(i.location::geometry) as latitude,
			ST_X(i.location::geometry) as longitude,
			COUNT(DISTINCT l.user_id) as user_count
		FROM incidents i
		LEFT JOIN location_checks l ON 
			ST_DWithin(i.location::geography, l.location::geography, i.radius_meters)
			AND l.created_at >= NOW() - ($1 * INTERVAL '1 second')
		WHERE i.is_active = TRUE
		GROUP BY i.id
		ORDER BY i.id ASC
	`

	rows, err := q.Query(ctx, query, window.Seconds())
	if err != nil {
		return nil, fmt.Errorf("failed to query incident stats: %w", err)
	}
	defer rows.Close()

	var stats []models.Stats

	for rows.Next() {
		var item models.Stats
		if err := rows.Scan(
			&item.IncidentID,
			&item.Latitude,
			&item.Longitude,
			&item.UserCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan stats item: %w", err)
		}
		stats = append(stats, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return stats, nil
}

func (r *Repo) GetActive(ctx context.Context) ([]models.IncidentShort, error) {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		SELECT 
			id, 
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude,
			radius_meters
		FROM incidents
		WHERE is_active = TRUE
		ORDER BY id ASC
	`

	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active incidents: %w", err)
	}
	defer rows.Close()

	shorts := make([]models.IncidentShort, 0)

	for rows.Next() {
		var item models.IncidentShort
		if err := rows.Scan(
			&item.ID,
			&item.Latitude,
			&item.Longitude,
			&item.Radius,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active incident: %w", err)
		}
		shorts = append(shorts, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return shorts, nil
}
