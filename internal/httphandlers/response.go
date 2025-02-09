package httphandlers

import (
	"encoding/json"
	"net/http"
	"sarabi/internal/misc"
)

const (
	authorizationHeader = "X-Access-Token"
)

type (
	response struct {
		Error   bool        `json:"error"`
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	r := response{
		Error:   false,
		Message: message,
		Data:    data,
	}
	b, _ := json.Marshal(r)
	w.Write(b)
}

func writeError(w http.ResponseWriter, errorCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errorCode)
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

func writeSSELine(w http.ResponseWriter, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, _ = w.Write(bytes)
	_, _ = w.Write(misc.Seperator)
	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}
	return nil
}
