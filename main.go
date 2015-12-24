package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gistia/slackbot/db"
	_ "github.com/gistia/slackbot/importer"
	"github.com/gistia/slackbot/robots"
	"github.com/gistia/slackbot/userbot"
	"github.com/gistia/slackbot/utils"
	"github.com/gistia/slackbot/web"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/yvasiyarov/gorelic"
)

func main() {
	r := mux.NewRouter()

	// command handlers
	r.HandleFunc("/slack", slashCommandHandler)
	r.HandleFunc("/slack_hook", hookHandler)
	r.HandleFunc("/callback", callbackHandler)

	// API endpoints
	r.HandleFunc("/api/projects", web.GetProjects)

	http.Handle("/", r)

	pokerRouter := mux.NewRouter()
	pokerRouter.Methods("GET").Path("/poker").HandlerFunc(web.NewPokerStories)
	pokerRouter.Methods("POST").Path("/poker").HandlerFunc(web.CreatePokerStories)
	http.Handle("/poker", pokerRouter)

	go db.SetupDB()
	if os.Getenv("RUN_BOT") != "" {
		go startBot()
	}
	userbot.InitActions()
	startServer()
}

func startNewRelic() {
	key := os.Getenv("NEW_RELIC_LICENSE_KEY")
	agent := gorelic.NewAgent()
	agent.Verbose = true
	agent.NewrelicLicense = key

	fmt.Println("Starting NewRelic for " + key)
	agent.Run()
}

func startBot() {
	userbot.Start()
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	d := schema.NewDecoder()
	command := new(robots.OutgoingWebHook)
	err = d.Decode(command, r.PostForm)
	if err != nil {
		log.Println("Couldn't parse post request:", err)
	}
	// if command.Text == "" || command.Token != os.Getenv(fmt.Sprintf("%s_OUT_TOKEN", strings.ToUpper(command.TeamDomain))) {
	// 	log.Printf("[DEBUG] Ignoring request from unidentified source: %s - %s", command.Token, r.Host)
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }
	fmt.Println(command.Text)
	fmt.Println(command.TriggerWord)
	com := strings.TrimPrefix(command.Text, command.TriggerWord) // +" ")
	c := strings.Split(com, " ")
	command.Robot = c[0]
	command.Text = strings.Join(c[1:], " ")

	robots := getRobots(command.Robot)
	if len(robots) == 0 {
		msg := fmt.Sprintf("No robot for %s (%s)", command.Robot, command.Text)
		jsonResp(w, msg)
		return
	}
	resp := ""
	for _, robot := range robots {
		resp += fmt.Sprintf("\n%s", robot.Run(&command.Payload))
	}
	w.WriteHeader(http.StatusOK)
	jsonResp(w, strings.TrimSpace(resp))
}

func slashCommandHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	d := schema.NewDecoder()
	command := new(robots.SlashCommand)
	err = d.Decode(command, r.PostForm)
	if err != nil {
		log.Println("Couldn't parse post request:", err)
	}
	if command.Command == "" || command.Token == "" {
		log.Printf("[DEBUG] Ignoring request from unidentified source: %s - %s", command.Token, r.Host)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	command.Robot = command.Command[1:]

	if token := os.Getenv(fmt.Sprintf("%s_SLACK_TOKEN", strings.ToUpper(command.Robot))); token != "" && token != command.Token {
		log.Printf("[DEBUG] Ignoring request from unidentified source: %s - %s", command.Token, r.Host)
		w.WriteHeader(http.StatusBadRequest)
	}
	robots := getRobots(command.Robot)
	if len(robots) == 0 {
		plainResp(w, "No robot for that command yet :(")
		return
	}
	resp := ""
	for _, robot := range robots {
		resp += fmt.Sprintf("\n%s", robot.Run(&command.Payload))
	}
	plainResp(w, strings.TrimSpace(resp))
}

type MvnAuthResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func mvnError(domain string, chanId string, user string, s string) {
	msg := fmt.Sprintf(
		"There was an error authenticating @%s with Mavenlink: %s",
		user, s)
	MvnSend(domain, chanId, msg)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	domain := params.Get("domain")
	user := params.Get("user")
	chanId := params.Get("channel")

	fmt.Println("callbackHandler", params)

	if params.Get("error") != "" {
		mvnError(domain, chanId, user, params.Get("error_description"))
		return
	}

	// client_id is the ID assigned to your application by Mavenlink
	// client_secret is the secret token assigned to your application by Mavenlink
	// grant_type must be set to "authorization_code" in order to exchange a code for an access token
	// code is the value that was returned in the code query parameter when Mavenlink redirected back to your redirect_uri
	// redirect_uri is the exact same value that you used in the original request to /oauth/authorize
	// 199c12d7ebc29800ec202fbad0b2585a1f84950536851527a95a024881adbb2b
	code := params.Get("code")

	callback := os.Getenv("MAVENLINK_CALLBACK")
	callback = fmt.Sprintf("%s?domain=%s&user=%s&channel=%s",
		callback, domain, user, chanId)

	rp := url.Values{}
	rp.Set("client_id", os.Getenv("MAVENLINK_APP_ID"))
	rp.Set("client_secret", os.Getenv("MAVENLINK_SECRET"))
	rp.Set("grant_type", "authorization_code")
	rp.Set("code", code)
	rp.Set("redirect_uri", callback)

	fmt.Println("Sending", rp)

	resp, err := utils.Request(
		"POST", "https://app.mavenlink.com/oauth/token",
		rp, map[string]string{})
	if err != nil {
		mvnError(domain, chanId, user, err.Error())
		return
	}

	fmt.Println("Response from Mavenlink:", string(resp))

	var b *MvnAuthResponse

	err = json.Unmarshal(resp, &b)
	if err != nil {
		fmt.Println("Error unmarshal", err.Error())
		mvnError(domain, chanId, user, err.Error())
		return
	}

	if b.Error != "" {
		mvnError(domain, chanId, user, b.ErrorDescription)
		return
	}

	err = db.SetSetting(user, "MAVENLINK_TOKEN", b.AccessToken)
	if err != nil {
		mvnError(domain, chanId, user, err.Error())
		return
	}

	msg := fmt.Sprintf("Saved Mavenlink authentication token for @%s", user)
	MvnSend(domain, chanId, msg)

	w.WriteHeader(http.StatusOK)
}

func MvnSend(domain string, chanId string, s string) {
	response := &robots.IncomingWebhook{
		Domain:      domain,
		Channel:     chanId,
		Username:    "Mavenlink Bot",
		Text:        s,
		IconEmoji:   ":chart_with_upwards_trend:",
		UnfurlLinks: true,
		Parse:       robots.ParseStyleFull,
	}

	response.Send()
}

func jsonResp(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := map[string]string{"text": msg}
	r, err := json.Marshal(resp)
	if err != nil {
		log.Println("Couldn't marshal hook response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(r)
}

func plainResp(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(msg))
}

func startServer() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT not set")
	}
	log.Printf("Starting HTTP server on %s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("Server start error: ", err)
	}
}

func getRobots(command string) []robots.Robot {
	fmt.Println("Robots:", robots.Robots)
	if r, ok := robots.Robots[command]; ok {
		return r
	}
	return nil
}
