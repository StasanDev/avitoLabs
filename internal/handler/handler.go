package handler

import (
	"net/http"

	api "github.com/StasanDev/avitoLabs/internal/generated"
)

type Handler struct {
	tripService tripService
}

func NewHandler(tripService tripService) *Handler {
	return &Handler{
		tripService: tripService,
	}
}

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	_ *http.Request,
	_ api.CreateTripParams,
) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	_ *http.Request,
	_ api.TripId,
) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	_ *http.Request,
	_ api.TripId,
) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) ListTripPositions(
	w http.ResponseWriter,
	_ *http.Request,
	_ api.TripId,
) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateTripPosition(
	w http.ResponseWriter,
	_ *http.Request,
	_ api.TripId,
) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) Ready(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
