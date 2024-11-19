package httphandlers

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func Routes(h *ApiHandler) chi.Router {
	r := chi.NewRouter()
	r.Route("/v1", func(rr chi.Router) {
		rr.Post("/applications", h.CreateApplication)
		rr.Put("/applications/{application_id}/envs", h.UpdateEnvs)
		rr.Post("/deploy", h.Deploy)
		rr.Get("/h", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("Hoi, we're HTTPs live!"))
		})
	})
	return r
}
