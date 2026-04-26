package api

import "net/http"

func (rt *router) liveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
