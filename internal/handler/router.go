package handler

import (
	"net/http"

	api "github.com/StasanDev/avitoLabs/internal/generated"
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler api.ServerInterface) http.Handler {
	r := chi.NewRouter()

	return api.HandlerWithOptions(handler, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			writeProblem(w, r, http.StatusBadRequest,
				"invalid_request", "Invalid request", "Request validation failed")
		},
	})
}
