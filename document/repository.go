package document

import (
	"fmt"
	"strings"
	"github.com/jmoiron/sqlx"
)

const (
	isQuerySearch = "(dd.document_name ILIKE $%d OR dd.staff ILIKE $%d OR dd.team ILIKE $%d)"
	isFilterDataType = "dd.data_type = $%d"
	isFilterCategoryType = "d.category = $%d"
	isFilterStatusType = "dd.status = $%d"
	isFilterCreateAt = "dd.created_at >= $%d"
	isFilterEndAt = "dd.created_at <= $%d"
	isQueryCreatedAt = "dd.created_at"
)

type DocumentRepository struct {
	db *sqlx.DB
}

func NewDocumentRepository(db *sqlx.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) CreateDocument(document *Document) error {
	query := `INSERT INTO documents (category) VALUES ($1) RETURNING id`
	return r.db.QueryRow(query, document.Category).Scan(&document.ID)
}

func (r *DocumentRepository) CreateDocumentDetail(detail *DocumentDetail) error {
	query := `
		INSERT INTO document_details 
		(document_id, document_name, filename, data_type, staff, team, status, is_latest, is_approve, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()) 
		RETURNING id, created_at
	`
	return r.db.QueryRow(
		query,
		detail.DocumentID,
		detail.DocumentName,
		detail.Filename,
		detail.DataType,
		detail.Staff,
		detail.Team,
		detail.Status,
		detail.IsLatest,
		detail.IsApprove,
	).Scan(&detail.ID, &detail.CreatedAt)
}

func (r *DocumentRepository) GetAllDocuments(filter DocumentFilter) ([]DocumentWithDetail, error) {
	
	conditions, args, argIndex := r.buildDocumentFilters(filter)

	base := `
		SELECT 
			d.id AS id,
			d.category AS category,
			dd.document_name AS document_name,
			dd.filename AS filename,
			dd.data_type AS data_type,
			dd.staff AS staff,
			dd.team AS team,
			dd.status AS status,
			dd.is_latest AS is_latest,
			dd.is_approve AS is_approve,
			dd.created_at AS created_at,
			dd.ingest_status AS ingest_status
		FROM documents d
		INNER JOIN document_details dd ON d.id = dd.document_id
		WHERE dd.is_latest = true
	`

	query := base
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	
	query += r.buildSortClause(filter)

	
	limit, offset := r.ensurePagination(filter.Limit, filter.Offset)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	
	var documents []DocumentWithDetail
	err := r.db.Select(&documents, query, args...)
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func (r *DocumentRepository) buildDocumentFilters(filter DocumentFilter) ([]string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(isQuerySearch, argIndex, argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if filter.DataType != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterDataType, argIndex))
		args = append(args, filter.DataType)
		argIndex++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterCategoryType, argIndex))
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterStatusType, argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.IngestStatus != "" {
		if filter.IngestStatus == "null" {
			conditions = append(conditions, "dd.ingest_status IS NULL")
		} else {
			conditions = append(conditions, fmt.Sprintf("dd.ingest_status = $%d", argIndex))
			args = append(args, filter.IngestStatus)
			argIndex++
		}
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterCreateAt, argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterEndAt, argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	return conditions, args, argIndex
}

func (r *DocumentRepository) buildSortClause(filter DocumentFilter) string {
	allowedSort := map[string]bool{isQueryCreatedAt: true, "dd.document_name": true, "dd.staff": true}
	sortBy := isQueryCreatedAt

	if filter.SortBy != "" {
		sb := filter.SortBy
		if allowedSort[sb] {
			sortBy = sb
		} else if allowedSort["dd."+sb] {
			sortBy = "dd." + sb
		}
	}

	dir := "DESC"
	if strings.ToUpper(filter.SortDirection) == "ASC" {
		dir = "ASC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, dir)
}

func (r *DocumentRepository) ensurePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (r *DocumentRepository) GetTotalDocuments(filter DocumentFilter) (int, error) {
	base := `
		SELECT COUNT(*)
		FROM documents d
		INNER JOIN document_details dd ON d.id = dd.document_id
		WHERE dd.is_latest = true
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(isQuerySearch, argIndex, argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if filter.DataType != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterDataType, argIndex))
		args = append(args, filter.DataType)
		argIndex++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterCategoryType, argIndex))
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterStatusType, argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.IngestStatus != "" {
		if filter.IngestStatus == "null" {
			conditions = append(conditions, "dd.ingest_status IS NULL")
		} else {
			conditions = append(conditions, fmt.Sprintf("dd.ingest_status = $%d", argIndex))
			args = append(args, filter.IngestStatus)
			argIndex++
		}
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterCreateAt, argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterEndAt, argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	query := base
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var total int
	err := r.db.QueryRow(query, args...).Scan(&total)
	return total, err
}

func (r *DocumentRepository) GetDocumentByID(id int) (*Document, error) {
	var document Document
	err := r.db.Get(&document, `SELECT id, category FROM documents WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &document, nil
}

func (r *DocumentRepository) GetDocumentDetailsByDocumentID(documentID int) ([]DocumentDetail, error) {
	var details []DocumentDetail
	query := `
		SELECT 
			id, document_id, document_name, filename, data_type, staff, team, 
			status, is_latest, is_approve, created_at, ingest_status
		FROM document_details
		WHERE document_id = $1
		ORDER BY created_at DESC
	`
	err := r.db.Select(&details, query, documentID)
	if err != nil {
		return nil, err
	}
	return details, nil
}

func (r *DocumentRepository) UpdateDocumentDetailLatest(documentID int) error {
	query := `UPDATE document_details SET is_latest = false WHERE document_id = $1`
	_, err := r.db.Exec(query, documentID)
	return err
}

func (r *DocumentRepository) UpdateDocumentDetailLatestByID(id int, isLatest bool) error {
	query := `UPDATE document_details SET is_latest = $1 WHERE id = $2`
	_, err := r.db.Exec(query, isLatest, id)
	return err
}

func (r *DocumentRepository) GetDocumentDetailByID(id int) (*DocumentDetail, error) {
	var detail DocumentDetail
	query := `
		SELECT 
			id, document_id, document_name, filename, data_type, staff, team, 
			status, is_latest, is_approve, created_at, ingest_status
		FROM document_details
		WHERE id = $1
	`
	err := r.db.Get(&detail, query, id)
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func (r *DocumentRepository) UpdateDocumentDetailApprove(id int, isApprove bool) error {
	query := `UPDATE document_details SET is_approve = $1 WHERE id = $2`
	_, err := r.db.Exec(query, isApprove, id)
	return err
}

func (r *DocumentRepository) UpdateAllDocumentDetailsApprove(documentID int, isApprove bool) error {
	query := `UPDATE document_details SET is_approve = $1 WHERE document_id = $2`
	_, err := r.db.Exec(query, isApprove, documentID)
	return err
}

func (r *DocumentRepository) UpdateDocumentDetailStatus(id int, status string) error {
	query := `UPDATE document_details SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *DocumentRepository) GetAllDocumentDetails(filter DocumentDetailFilter) ([]DocumentDetail, error) {
	
	conditions, args, argIndex := r.buildDocumentDetailFilters(filter)

	base := `
		SELECT 
			dd.id, dd.document_id, dd.document_name, dd.filename, dd.data_type, 
			dd.staff, dd.team, dd.status, dd.is_latest, dd.is_approve, dd.created_at,
			d.category,
			dd.ingest_status
		FROM document_details dd
		INNER JOIN documents d ON dd.document_id = d.id
		WHERE 1=1
	`

	query := base
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	
	query += r.buildDetailSortClause(filter)

	
	limit, offset := r.ensurePagination(filter.Limit, filter.Offset)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	
	var details []DocumentDetail
	err := r.db.Select(&details, query, args...)
	if err != nil {
		return nil, err
	}
	return details, nil
}






func (r *DocumentRepository) buildDocumentDetailFilters(filter DocumentDetailFilter) ([]string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(isQuerySearch, argIndex, argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if filter.DataType != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterDataType, argIndex))
		args = append(args, filter.DataType)
		argIndex++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterCategoryType, argIndex))
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterStatusType, argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.DocumentName != "" {
		conditions = append(conditions, fmt.Sprintf("dd.document_name ILIKE $%d", argIndex))
		args = append(args, "%"+filter.DocumentName+"%")
		argIndex++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterCreateAt, argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterEndAt, argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	return conditions, args, argIndex
}

func (r *DocumentRepository) buildDetailSortClause(filter DocumentDetailFilter) string {
	allowedSort := map[string]bool{isQueryCreatedAt: true, "dd.document_name": true, "dd.staff": true}
	sortBy := isQueryCreatedAt

	if filter.SortBy != "" {
		sb := filter.SortBy
		if allowedSort[sb] {
			sortBy = sb
		} else if allowedSort["dd."+sb] {
			sortBy = "dd." + sb
		}
	}

	dir := "DESC"
	if strings.ToUpper(filter.SortDirection) == "ASC" {
		dir = "ASC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, dir)
}

func (r *DocumentRepository) GetTotalDocumentDetails(filter DocumentDetailFilter) (int, error) {
	base := `
		SELECT COUNT(*)
		FROM document_details dd
		INNER JOIN documents d ON dd.document_id = d.id
		WHERE 1=1
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(isQuerySearch, argIndex, argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if filter.DataType != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterDataType, argIndex))
		args = append(args, filter.DataType)
		argIndex++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterCategoryType, argIndex))
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf(isFilterStatusType, argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.DocumentName != "" {
		conditions = append(conditions, fmt.Sprintf("dd.document_name ILIKE $%d", argIndex))
		args = append(args, "%"+filter.DocumentName+"%")
		argIndex++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterCreateAt, argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf(isFilterEndAt, argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	query := base
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var total int
	err := r.db.QueryRow(query, args...).Scan(&total)
	return total, err
}

func (r *DocumentRepository) DeleteDocument(id int) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *DocumentRepository) DeleteDocumentDetails(documentID int) error {
	query := `DELETE FROM document_details WHERE document_id = $1`
	_, err := r.db.Exec(query, documentID)
	return err
}

func (r *DocumentRepository) GetTeamNameByUserID(userID int64) (string, error) {
	var teamName string
	query := `
		SELECT t.name
		FROM teams t
		JOIN roles r ON r.team_id = t.id
		JOIN users u ON u.role_id = r.id
		WHERE u.id = $1
	`
	err := r.db.Get(&teamName, query, userID)
	if err != nil {
		return "", err
	}
	return teamName, nil
}

func (r *DocumentRepository) UpdateDocumentDetailIngestStatus(id int, status string) error {
	query := `UPDATE document_details SET ingest_status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *DocumentRepository) GetApprovedLatestDocumentDetailByDocumentID(documentID int) (*DocumentDetail, error) {
	var detail DocumentDetail
	query := `
		SELECT 
			id, document_id, document_name, filename, data_type, staff, team, 
			status, is_latest, is_approve, created_at, ingest_status
		FROM document_details
		WHERE document_id = $1 AND is_latest = true AND is_approve = true
		LIMIT 1
	`
	err := r.db.Get(&detail, query, documentID)
	if err != nil {
		return nil, err
	}
	return &detail, nil
}


func (r *DocumentRepository) GetLatestDetailByDocumentName(docName string) (*DocumentDetail, error) {
	var detail DocumentDetail
	
	query := `
		SELECT 
			id, document_id, document_name, filename, data_type, staff, team, 
			status, is_latest, is_approve, created_at, ingest_status
		FROM document_details
		WHERE document_name = $1 AND is_latest = true
		LIMIT 1
	`
	err := r.db.Get(&detail, query, docName)
	if err != nil {
		return nil, err
	}
	return &detail, nil
}


func (r *DocumentRepository) DeleteDocumentDetailHard(id int) error {
	_, err := r.db.Exec(`DELETE FROM document_details WHERE id = $1`, id)
	return err
}

func (r *DocumentRepository) CheckDuplicationFileByDocumentName(docName string) (*DocumentDetail, error) {
	var detail DocumentDetail
	
	query := `
		SELECT id, document_name 
		FROM document_details
		WHERE document_name = $1 AND is_latest = true
		LIMIT 1
	`
	err := r.db.Get(&detail, query, docName)
	if err != nil {
		return nil, err
	}
	return &detail, nil
}