package web

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/gistia/slackbot/db"
	"github.com/gistia/slackbot/utils"
	"github.com/gorilla/sessions"
)

var Store = sessions.NewCookieStore([]byte("6051dfc4b4602d7d8261f51c3bea2ad941a4167a753e5772ee2ed45c5959"))

type PokerPage struct {
	SessionTitle string
	ChannelName  string
	ChannelID    string
	Message      string
	Stories      []db.PokerStory
}

func NewPokerStories(w http.ResponseWriter, r *http.Request) {
	webSession, _ := Store.Get(r, "session")
	channel := q(r.URL, "channel")
	channelId := q(r.URL, "channel_id")

	session, err := db.GetCurrentSession(channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if session == nil {
		fmt.Fprintf(w, "No current session for channel %s", channel)
		return
	}

	stories, err := session.GetStories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t, err := template.ParseFiles(
		"web/public/index.html", "web/public/poker/new.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := &PokerPage{
		SessionTitle: session.Title,
		ChannelName:  channel,
		ChannelID:    channelId,
		Stories:      stories,
		Message:      "",
	}

	if webSession.Values["message"] != nil {
		page.Message = webSession.Values["message"].(string)
	}
	webSession.Values["message"] = ""
	webSession.Save(r, w)

	t.Execute(w, page)
	return
}

func CreatePokerStories(w http.ResponseWriter, r *http.Request) {
	webSession, _ := Store.Get(r, "session")
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	channel := r.PostFormValue("channel")
	channelId := r.PostFormValue("channel_id")
	stories := r.PostFormValue("stories")

	session, err := db.GetCurrentSession(channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if session == nil {
		fmt.Fprintf(w, "No current session for channel %s", channel)
		return
	}

	lines := strings.Split(stories, "\n")
	for _, s := range lines {
		s = strings.Trim(s, "\r")
		session.StartPokerStory(s)
	}

	h := utils.NewSlackHandler("poker", ":game_die:")
	h.SendMsg(channelId,
		fmt.Sprintf("%d stories were added to be estimated", len(lines)))

	msg := fmt.Sprintf("Added %d stories to this session", len(lines))
	webSession.Values["message"] = msg
	webSession.Save(r, w)

	location := fmt.Sprintf("/poker?channel_id=%s&channel=%s", channelId, channel)
	http.Redirect(w, r, location, 301)
}
