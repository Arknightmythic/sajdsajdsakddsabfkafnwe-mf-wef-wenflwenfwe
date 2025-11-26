package guide

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type GuideRepository struct {
	db *sqlx.DB
}

func NewGuideRepository(db *sqlx.DB) *GuideRepository {
	return &GuideRepository{db: db}
}

func (r *GuideRepository) Create(guide *Guide) error {
	query := `
		INSERT INTO guides (title, description, filename, original_filename, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(
		query,
		guide.Title,
		guide.Description,
		guide.Filename,
		guide.OriginalFilename,
	).Scan(&guide.ID, &guide.CreatedAt, &guide.UpdatedAt)
}

func (r *GuideRepository) GetAll(filter GuideFilter) ([]Guide, int, error) {
	var guides []Guide
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM guides %s", where)
	var total int
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	allowedSort := map[string]bool{"created_at": true, "title": true, "updated_at": true}
	sortBy := "created_at"
	if filter.SortBy != "" && allowedSort[filter.SortBy] {
		sortBy = filter.SortBy
	}

	sortDirection := "DESC"
	if strings.ToUpper(filter.SortDirection) == "ASC" {
		sortDirection = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, title, description, filename, original_filename, created_at, updated_at
		FROM guides
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortBy, sortDirection, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	if err := r.db.Select(&guides, query, args...); err != nil {
		return nil, 0, err
	}

	return guides, total, nil
}

func (r *GuideRepository) GetByID(id int) (*Guide, error) {
	var guide Guide
	query := `SELECT * FROM guides WHERE id = $1`
	err := r.db.Get(&guide, query, id)
	if err != nil {
		return nil, err
	}
	return &guide, nil
}

func (r *GuideRepository) Update(guide *Guide) error {
	query := `
		UPDATE guides 
		SET title = $1, description = $2, filename = $3, original_filename = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`
	return r.db.QueryRow(
		query,
		guide.Title,
		guide.Description,
		guide.Filename,
		guide.OriginalFilename,
		guide.ID,
	).Scan(&guide.UpdatedAt)
}

func (r *GuideRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM guides WHERE id = $1`, id)
	return err
}
