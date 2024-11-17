package httphandlers

import (
	"encoding/json"
	"net/http"
)

type (
	response struct {
		Error   bool
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}
)

func badRequest(w http.ResponseWriter, err error) {
	writeError(w, http.StatusBadRequest, err)
}

func serverError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusInternalServerError, err)
}

func unauthorized(w http.ResponseWriter, err error) {
	writeError(w, http.StatusUnauthorized, err)
}

func forbidden(w http.ResponseWriter, err error) {
	writeError(w, http.StatusForbidden, err)
}

func ok(w http.ResponseWriter, message string, data interface{}) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	r := response{
		Error:   false,
		Message: message,
		Data:    data,
	}
	b, _ := json.Marshal(r)
	w.Write(b)
}

func writeError(w http.ResponseWriter, errorCode int, err error) {
	w.WriteHeader(errorCode)
	w.Header().Set("Content-Type", "application/json")
	errmsg := ""
	if err != nil {
		errmsg = err.Error()
	}

	r := response{
		Error:   true,
		Message: errmsg,
	}
	data, _ := json.Marshal(r)
	w.Write(data)
}
