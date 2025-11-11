package team

import (
	"dokuprime-be/util"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TeamHandler struct {
	service *TeamService
}

func NewTeamHandler(service *TeamService) *TeamHandler {
	return &TeamHandler{service: service}
}

func (h *TeamHandler) CreateTeam(ctx *gin.Context) {
	var req struct {
		Name  string   `json:"name"`
		Pages []string `json:"pages"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	team := &Team{
		Name:  req.Name,
		Pages: req.Pages,
	}

	if err := h.service.Create(team); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(ctx, "Team created successfully", team)
}

func (h *TeamHandler) GetAll(ctx *gin.Context) {

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	search := ctx.DefaultQuery("search", "")

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	teams, total, err := h.service.GetAll(limit, offset, search)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"teams":  teams,
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"search": search,
	}

	util.SuccessResponse(ctx, "Teams retrieved successfully", response)
}

func (h *TeamHandler) GetTeamByID(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid team ID")
		return
	}

	team, err := h.service.GetByID(id)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusNotFound, "Team not found")
		return
	}

	util.SuccessResponse(ctx, "Team retrieved successfully", team)
}

func (h *TeamHandler) UpdateTeam(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid team ID")
		return
	}

	var req struct {
		Name  string   `json:"name"`
		Pages []string `json:"pages"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	team := &Team{
		ID:    id,
		Name:  req.Name,
		Pages: req.Pages,
	}

	if err := h.service.Update(team); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Team updated successfully", team)
}

func (h *TeamHandler) DeleteTeam(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid team ID")
		return
	}

	if err := h.service.Delete(id); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Team deleted successfully", nil)
}
