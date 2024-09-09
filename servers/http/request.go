package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type H map[string]any

func BindJson[T any](r *http.Request, v T) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		return err
	}
	return nil
}

func ShouldBindJson[T any](r *http.Request, v T) error {
	err := BindJson(r, v)
	if err != nil {
		return err
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(v)
	if err != nil {
		return err
	}
	return nil
}

func Json[T any](status int, w http.ResponseWriter, v T) error {
	w.Header().Set("Content-Type", "application/json")
	Status(status, w)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return err
	}
	return nil
}

func Status(status int, w http.ResponseWriter) {
	w.WriteHeader(status)
}

func GroupRoutes(enging *http.ServeMux)
