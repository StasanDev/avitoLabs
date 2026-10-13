package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/StasanDev/avitoLabs/internal/domain"
	api "github.com/StasanDev/avitoLabs/internal/generated"
)

type Handler struct {
	tripService      tripService
	pinger           pinger
	readinessTimeout time.Duration
}

func NewHandler(
	tripService tripService,
	pinger pinger,
	readinessTimeout time.Duration,
) *Handler {
	return &Handler{
		tripService:      tripService,
		pinger:           pinger,
		readinessTimeout: readinessTimeout,
	}
}

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	params api.CreateTripParams,
) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	var body api.TripData
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	if err := validateRequiredTripFields(payload); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	if err := validateTripData(body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	tripInput := domain.TripInput{
		UserID:   body.UserId,
		DriverID: body.DriverId,
		StartPoint: domain.Coordinates{
			Latitude:  body.StartPoint.Latitude,
			Longitude: body.StartPoint.Longitude,
		},
		EndPoint: domain.Coordinates{
			Latitude:  body.EndPoint.Latitude,
			Longitude: body.EndPoint.Longitude,
		},
		Price: body.Price,
	}

	if params.IdempotencyKey != nil {
		hash := sha256.Sum256(payload)
		tripInput.IdempotencyKey = params.IdempotencyKey
		tripInput.RequestHash = hash[:]
	}

	trip, replayed, err := h.tripService.CreateTrip(r.Context(), tripInput)
	if err != nil {
		if errors.Is(err, domain.ErrDriverBusy) {
			writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
			return
		}
		if errors.Is(err, domain.ErrIdempotencyConflict) {
			writeProblem(
				w, r, http.StatusConflict, "idempotency_conflict", "Idempotency conflict",
				"Idempotency-Key was already used with a different request body",
			)
			return
		}

		log.Printf("create trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		return
	}

	if replayed {
		writeJSON(w, http.StatusOK, tripToAPI(trip))
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, http.StatusCreated, tripToAPI(trip))
}

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	id api.TripId,
) {
	trip, err := h.tripService.GetTrip(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTripNotFound) {
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "no trip with this id")
			return
		}
		log.Printf("get trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "unexpected error")
		return
	}
	writeJSON(w, http.StatusOK, tripToAPI(trip))
}

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripId api.TripId,
) {
	trip, err := h.tripService.FinishTrip(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, domain.ErrTripCompleted) {
			writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "trip with this id is already done")
			return
		}
		if errors.Is(err, domain.ErrTripNotFound) {
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "no trip with this id")
			return
		}
		log.Printf("finish trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "unexpected error")
		return
	}
	writeJSON(w, http.StatusOK, tripToAPI(trip))
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.readinessTimeout)
	defer cancel()

	if err := h.pinger.Ping(ctx); err != nil {
		log.Printf("readiness check failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}

	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

// Не используется в 1 лабораторной
func (h *Handler) ListTripPositions(
	w http.ResponseWriter,
	_ *http.Request,
	_ api.TripId,
) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Не используется в 1 лабораторной
func (h *Handler) CreateTripPosition(
	w http.ResponseWriter,
	_ *http.Request,
	_ api.TripId,
) {
	w.WriteHeader(http.StatusNotImplemented)
}
