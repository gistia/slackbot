package web

import (
	"encoding/json"
	"net/http"

	"github.com/gistia/slackbot/db"
)

func GetProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := db.GetProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(projects)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
