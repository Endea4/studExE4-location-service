package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Endea4/studExE4-location-service/internal/model"
	"github.com/Endea4/studExE4-location-service/internal/repository"
	"github.com/go-chi/chi/v5"
)

type LocationHandler struct {
	repo *repository.LocationRepository
}

func NewLocationHandler(repo *repository.LocationRepository) *LocationHandler {
	return &LocationHandler{repo: repo}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.ErrorResponse{Error: msg, Code: status})
}

// UpdateLocation godoc
// @Summary Update entity location
// @Description Update an entity's current GPS position. Called every 2-5 seconds by clients.
// @Tags location
// @Accept json
// @Produce json
// @Param body body model.LocationUpdate true "Location data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /location [post]
func (h *LocationHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var req model.LocationUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RefID == "" {
		writeError(w, http.StatusBadRequest, "ref_id is required")
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		writeError(w, http.StatusBadRequest, "latitude must be between -90 and 90")
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		writeError(w, http.StatusBadRequest, "longitude must be between -180 and 180")
		return
	}

	if err := h.repo.UpdateLocation(r.Context(), &req); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update location")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"ref_id": req.RefID,
	})
}

// GetLocation godoc
// @Summary Get entity location
// @Description Get an entity's current GPS position
// @Tags location
// @Produce json
// @Param ref_id path string true "Reference ID"
// @Success 200 {object} model.TrackedEntity
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /location/{ref_id} [get]
func (h *LocationHandler) GetLocation(w http.ResponseWriter, r *http.Request) {
	refID := chi.URLParam(r, "ref_id")
	if refID == "" {
		writeError(w, http.StatusBadRequest, "ref_id is required")
		return
	}

	loc, err := h.repo.GetLocation(r.Context(), refID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get location")
		return
	}
	if loc == nil {
		writeError(w, http.StatusNotFound, "entity location not found")
		return
	}

	writeJSON(w, http.StatusOK, loc)
}

// FindNearby godoc
// @Summary Find nearby entities
// @Description Find all entities within a given radius of a point
// @Tags location
// @Produce json
// @Param lat query number true "Latitude"
// @Param lng query number true "Longitude"
// @Param radius query number false "Radius in km" default(5)
// @Success 200 {array} model.NearbyEntity
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /location/nearby [get]
func (h *LocationHandler) FindNearby(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	radiusStr := r.URL.Query().Get("radius")

	if latStr == "" || lngStr == "" {
		writeError(w, http.StatusBadRequest, "lat and lng query params are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid latitude")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid longitude")
		return
	}

	radius := 5.0
	if radiusStr != "" {
		radius, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil || radius <= 0 {
			writeError(w, http.StatusBadRequest, "invalid radius")
			return
		}
	}

	entities, err := h.repo.FindNearby(r.Context(), &model.NearbyQuery{
		Latitude:  lat,
		Longitude: lng,
		RadiusKm:  radius,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search nearby")
		return
	}

	if entities == nil {
		entities = []model.NearbyEntity{}
	}
	writeJSON(w, http.StatusOK, entities)
}

// GetAllLocations godoc
// @Summary Get all tracked locations
// @Description Get current positions of all tracked entities
// @Tags location
// @Produce json
// @Success 200 {array} model.TrackedEntity
// @Failure 500 {object} model.ErrorResponse
// @Router /location [get]
func (h *LocationHandler) GetAllLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := h.repo.GetAllLocations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get locations")
		return
	}
	if locations == nil {
		locations = []model.TrackedEntity{}
	}
	writeJSON(w, http.StatusOK, locations)
}

// RemoveEntity godoc
// @Summary Remove entity location
// @Description Remove an entity's location from the tracking system
// @Tags location
// @Produce json
// @Param ref_id path string true "Reference ID"
// @Success 200 {object} map[string]string
// @Failure 500 {object} model.ErrorResponse
// @Router /location/{ref_id} [delete]
func (h *LocationHandler) RemoveEntity(w http.ResponseWriter, r *http.Request) {
	refID := chi.URLParam(r, "ref_id")
	if refID == "" {
		writeError(w, http.StatusBadRequest, "ref_id is required")
		return
	}

	if err := h.repo.RemoveEntity(r.Context(), refID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove entity")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "removed",
		"ref_id": refID,
	})
}
