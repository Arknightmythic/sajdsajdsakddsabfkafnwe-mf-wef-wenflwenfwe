package grafana

type GenerateEmbedRequest struct {
	Category string `json:"category" binding:"required"`
	StartDate string `json:"start_date"`
	EndDate string `json:"end_date"`
}