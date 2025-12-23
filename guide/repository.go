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
		placeholder := "$" + fmt.Sprint(argIdx)
		conditions = append(conditions, "(title ILIKE "+placeholder+" OR description ILIKE "+placeholder+")")
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM guides " + where
	var total int
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Validate sort column
	allowedSort := map[string]bool{"created_at": true, "title": true, "updated_at": true}
	sortBy := "created_at"
	if filter.SortBy != "" && allowedSort[filter.SortBy] {
		sortBy = filter.SortBy
	}

	// Validate sort direction
	sortDirection := "DESC"
	if strings.ToUpper(filter.SortDirection) == "ASC" {
		sortDirection = "ASC"
	}

	limitPlaceholder := "$" + fmt.Sprint(argIdx)
	offsetPlaceholder := "$" + fmt.Sprint(argIdx+1)

	query := `SELECT id, title, description, filename, original_filename, created_at, updated_at
		FROM guides ` + where + `
		ORDER BY ` + sortBy + ` ` + sortDirection + `
		LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder

	args = append(args, filter.Limit, filter.Offset)

	if err := r.db.Select(&guides, query, args...); err != nil {
		return nil, 0, err
	}

	return guides, total, nil
}

func (r *GuideRepository) GetByID(id int) (*Guide, error) {
	var guide Guide
	query := `SELECT id, title, description, filename, original_filename, created_at, updated_at FROM guides WHERE id = $1`
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