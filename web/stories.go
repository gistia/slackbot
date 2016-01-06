package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/gistia/slackbot/db"
	"github.com/gistia/slackbot/robots/project"
	"github.com/gistia/slackbot/utils"
)

type StoryPage struct {
	Project     string
	ChannelName string
	ChannelID   string
	User        string
	Message     string
}

func NewStories(w http.ResponseWriter, r *http.Request) {
	webSession, _ := Store.Get(r, "session")
	user := q(r.URL, "user")
	channel := q(r.URL, "channel")
	channelID := q(r.URL, "channel_id")
	project := q(r.URL, "project")

	t, err := template.ParseFiles(
		"web/public/index.html", "web/public/stories/new.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := &StoryPage{
		ChannelName: channel,
		ChannelID:   channelID,
		Project:     project,
		User:        user,
		Message:     "",
	}

	if webSession.Values["message"] != nil {
		page.Message = webSession.Values["message"].(string)
	}
	webSession.Values["message"] = ""
	webSession.Save(r, w)

	t.Execute(w, page)
	return
}

func CreateStories(w http.ResponseWriter, r *http.Request) {
	webSession, _ := Store.Get(r, "session")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := r.PostFormValue("user")
	channel := r.PostFormValue("channel")
	channelID := r.PostFormValue("channel_id")
	stories := r.PostFormValue("stories")
	name := r.PostFormValue("project")

	p, err := db.GetProjectByName(name)
	if p == nil {
		fmt.Fprintf(w, "Project %s not found", channel)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lines := strings.Split(stories, "\n")
	for _, s := range lines {
		s = strings.Trim(s, "\r")
		_, _, err := robots.CreateStory(user, name, s, "feature")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	h := utils.NewSlackHandler("project", ":books:")
	h.SendMsg(channelID,
		fmt.Sprintf("%d stories were added to *%s* project", len(lines), name))

	msg := fmt.Sprintf("Added %d stories to the project", len(lines))
	webSession.Values["message"] = msg
	webSession.Save(r, w)

	location := fmt.Sprintf("/addstories?user=%s&channel_id=%s&channel=%s&project=%s",
		user, channelID, channel, name)
	http.Redirect(w, r, location, 301)
}

func q(url *url.URL, s string) string {
	item := url.Query()[s]
	if item == nil {
		return ""
	}

	return strings.Join(item, "")
}
