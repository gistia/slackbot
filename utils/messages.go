package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gistia/slackbot/robots"
)

type SlackHandler struct {
	BotName string
	Icon    string
	IconUrl string
}

func NewSlackHandler(name string, icon string) SlackHandler {
	if strings.HasPrefix(icon, "http://") || strings.HasPrefix(icon, "https://") {
		return SlackHandler{BotName: name, IconUrl: icon}
	}
	return SlackHandler{BotName: name, Icon: icon}
}

func (sh SlackHandler) Send(p *robots.Payload, s string) {
	sh.SendWithAttachments(p, s, nil)
}

func (sh SlackHandler) DirectSend(p *robots.Payload, s string) {
	msg := fmt.Sprintf("@%s: %s", p.UserName, s)
	sh.SendWithAttachments(p, msg, nil)
}

func (sh SlackHandler) SendError(p *robots.Payload, err error) {
	msg := fmt.Sprintf("Error in %s: %s\n", sh.BotName, err.Error())
	sh.Send(p, msg)
}

func (sh SlackHandler) SendWithAttachments(p *robots.Payload, s string, atts []robots.Attachment) {
	fmt.Println(p.TeamDomain)
	fmt.Println(" ->", s)
	for _, a := range atts {
		if a.TitleLink != "" {
			fmt.Println("    A:", a.Title, " -> ", a.TitleLink)
		} else {
			fmt.Println("    A:", a.Title)
		}
		fmt.Println("    T:", a.Text)
	}
	response := &robots.IncomingWebhook{
		Parse:       robots.ParseStyleFull,
		Domain:      p.TeamDomain,
		Channel:     p.ChannelID,
		Username:    sh.BotName,
		UnfurlLinks: true,
		Attachments: atts,
		Text:        s,
	}

	if sh.Icon != "" {
		response.IconEmoji = sh.Icon
	} else {
		response.IconURL = sh.IconUrl
	}

	response.Send()
}

func (sh SlackHandler) SendMsg(channel, s string) {
	domain := os.Getenv("SLACK_TEAM_DOMAIN")
	response := &robots.IncomingWebhook{
		Parse:       robots.ParseStyleFull,
		Domain:      domain,
		Channel:     channel,
		Username:    sh.BotName,
		UnfurlLinks: true,
		Text:        s,
	}

	if sh.Icon != "" {
		response.IconEmoji = sh.Icon
	} else {
		response.IconURL = sh.IconUrl
	}

	response.Send()
}

func FmtAttachment(fallback, title, url, text string) robots.Attachment {
	a := robots.Attachment{}
	a.Color = "#7CD197"
	a.Fallback = fallback
	a.Title = title
	a.TitleLink = url
	if text != "" {
		a.Text = text
	}
	return a
}

func StripUser(msg string) string {
	reg := regexp.MustCompile("(.*)<@.*?>:? ?(.*)")
	return reg.ReplaceAllString(msg, "${1}${2}")
}
