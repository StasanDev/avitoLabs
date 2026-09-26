package handler

import (
	"net/http"

	api "github.com/StasanDev/avitoLabs/internal/generated"
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler api.ServerInterface) http.Handler {
	r := chi.NewRouter()

	
	return api.HandlerFromMux(handler, r)
}
