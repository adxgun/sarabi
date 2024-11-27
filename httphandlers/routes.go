package httphandlers

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func Routes(h *ApiHandler) chi.Router {
	r := chi.NewRouter()
	r.Route("/v1", func(rr chi.Router) {
		rr.Post("/applications", h.CreateApplication)
		rr.Post("/deploy", h.Deploy)
		rr.Put("/applications/{application_id}/variables", h.UpdateVariables)
		rr.Patch("/applications/rollback", h.Rollback)
		rr.Patch("/applications/{application_id}/scale", h.Scale)
		rr.Post("/applications/{application_id}/domains", h.AddDomain)
		rr.Delete("/applications/{application_id}/domains", h.RemoveDomain)
		rr.Post("/applications/add-credentials", h.AddCredentials)

		rr.Get("/h", func(writer http.ResponseWriter, request *http.Request) {
			ok(writer, "Hoi, we're HTTPs live!", struct{}{})
		})
	})
	return r
}
