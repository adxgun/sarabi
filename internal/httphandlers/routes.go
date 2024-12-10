package httphandlers

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func Routes(h *ApiHandler) chi.Router {
	router := chi.NewRouter()
	router.Route("/v1", func(r chi.Router) {
		r.Post("/applications", h.CreateApplication)
		r.Post("/deploy", h.Deploy)
		r.Put("/applications/{application_id}/variables", h.UpdateVariables)
		r.Get("/applications/{application_id}/variables", h.ListVariables)
		r.Patch("/applications/rollback", h.Rollback)
		r.Patch("/applications/{application_id}/scale", h.Scale)
		r.Put("/applications/{application_id}/domains", h.AddDomain)
		r.Delete("/applications/{application_id}/domains", h.RemoveDomain)
		r.Post("/applications/add-credentials", h.AddCredentials)
		r.Get("/backups/{id}/download", h.DownloadBackup)
		r.Get("/applications/{application_id}/backups", h.ListBackups)
		r.Get("/applications/{application_id}/deployments", h.ListDeployments)
		r.Get("/applications", h.ListApplications)
		r.Get("/application", h.GetApplication)
		r.Post("/applications/{application_id}/destroy", h.Destroy)

		r.Get("/h", func(writer http.ResponseWriter, request *http.Request) {
			ok(writer, "Hoi, we're HTTPs live!", struct{}{})
		})
	})
	return router
}
