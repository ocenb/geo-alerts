package location

import (
	"context"
	"fmt"

	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/storage/transactor"
)

type Repo struct {
	tm *transactor.Manager
}

func New(tm *transactor.Manager) *Repo {
	return &Repo{tm}
}

func (r *Repo) SaveCheckLog(ctx context.Context, check *models.CheckLocationResult) error {
	q := r.tm.GetQueryEngine(ctx)

	query := `
		INSERT INTO location_checks (user_id, location, has_danger, created_at)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4, NOW())
	`
	_, err := q.Exec(ctx, query, check.UserID, check.Longitude, check.Latitude, check.HasDanger)
	if err != nil {
		return fmt.Errorf("failed to save check log: %w", err)
	}
	return nil
}
