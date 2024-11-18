package handlers

import (
	"net/http"

	"github.com/rs/zerolog"
)

type HandlerOptions struct {
	Log    *zerolog.Logger
	Client *http.Client
}

type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
