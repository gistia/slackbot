package robots

import (
	"errors"
	"fmt"

	"github.com/gistia/slackbot/db"
	"github.com/gistia/slackbot/pivotal"
	"github.com/gistia/slackbot/robots"
	"github.com/gistia/slackbot/utils"
)

type bot struct {
	handler utils.SlackHandler
}

func init() {
	handler := utils.NewSlackHandler("Pivotal", ":triangular_ruler:")
	s := &bot{handler: handler}
	robots.RegisterRobot("pvt", s)
}

func (r bot) Run(p *robots.Payload) string {
	go r.DeferredAction(p)
	return ""
}

func (r bot) DeferredAction(p *robots.Payload) {
	ch := utils.NewCmdHandler(p, r.handler, "pvt")
	ch.Handle("projects", r.sendProjects)
	ch.Handle("stories", r.sendStories)
	ch.Handle("mystories", r.sendMyStories)
	ch.Handle("auth", r.sendAuth)
	ch.Handle("users", r.users)

	cmds := []string{"start", "unstart", "finish", "accept", "reject", "deliver"}
	ch.HandleMany(cmds, r.setStoryState)

	ch.Process(p.Text)
}

func (r bot) users(p *robots.Payload, cmd utils.Command) error {
	projectID := cmd.Arg(0)
	if projectID == "" {
		r.handler.Send(p, "Missing project id. Use !pvt users <project-id>")
		return nil
	}

	pvt, err := conn(p.UserName)
	if err != nil {
		return err
	}

	project, err := pvt.GetProject(projectID)
	if err != nil {
		return err
	}
	if project == nil {
		r.handler.Send(p, "Project with id "+projectID+" doesn't exist.")
		return nil
	}

	memberships, err := pvt.GetProjectMemberships(projectID)
	if err != nil {
		return err
	}

	s := "Current users for project *" + project.Name + "*:\n"
	for _, m := range memberships {
		pp := m.Person
		s += fmt.Sprintf("%d - %s (%s)\n", pp.Id, pp.Name, pp.Email)
	}

	r.handler.Send(p, s)
	return nil
}

func (r bot) sendProjects(payload *robots.Payload, cmd utils.Command) error {
	var ps []pivotal.Project
	var err error

	term := cmd.Arg(0)

	pvt, err := conn(payload.UserName)
	if err != nil {
		return err
	}
	s := "Projects"

	if len(term) > 0 {
		fmt.Printf("Retrieving projects with term \"%s\"...\n\n", term)
		s += fmt.Sprintf(" matching '%s':\n", term)
		// ps, err = pvt.SearchProject(term)
	} else {
		s += ":\n"
		fmt.Println("Retrieving projects...")
		ps, err = pvt.Projects()
	}

	if err != nil {
		msg := fmt.Sprintf("Error: %s", err.Error())
		r.handler.Send(payload, msg)
		return nil
	}

	r.handler.Send(payload, s+projectTable(ps))
	return nil
}

func (r bot) sendStories(p *robots.Payload, cmd utils.Command) error {
	project := cmd.Arg(0)
	pvt, err := conn(p.UserName)
	if err != nil {
		return err
	}

	stories, err := pvt.Stories(project)
	if err != nil {
		return err
	}

	str := ""
	for _, s := range stories {
		str += fmt.Sprintf("%d - %s\n", s.Id, s.Name)
	}

	r.handler.Send(p, str)
	return nil
}

func (r bot) sendMyStories(p *robots.Payload, cmd utils.Command) error {
	args, err := cmd.ParseArgs("pvt-project-id")
	if err != nil {
		return err
	}

	project := args[0]
	pvt, err := conn(p.UserName)
	if err != nil {
		return err
	}

	user, err := db.GetUserByName(p.UserName)
	if err != nil {
		return err
	}

	filter := map[string]string{
		"owned_by": user.StrPivotalId(),
		"state":    "started,finished,delivered",
	}
	stories, err := pvt.FilteredStories(project, filter)
	if err != nil {
		return err
	}

	if len(stories) < 1 {
		r.handler.Send(p, "No open stories for *"+p.UserName+"*")
		return nil
	}

	str := "Current stories for *" + p.UserName + "*:\n"
	atts := []robots.Attachment{}
	for _, s := range stories {
		fallback := fmt.Sprintf("%d - %s - %s\n", s.Id, s.Name, s.State)
		title := fmt.Sprintf("%d - %s\n", s.Id, s.Name)
		a := utils.FmtAttachment(fallback, title, s.Url, s.State)
		atts = append(atts, a)
	}

	r.handler.SendWithAttachments(p, str, atts)
	return nil
}

func (r bot) setStoryState(p *robots.Payload, cmd utils.Command) error {
	state := cmd.Command
	id := cmd.Arg(0)
	pvt, err := conn(p.UserName)
	if err != nil {
		return err
	}

	user, err := db.GetUserByName(p.UserName)
	if err != nil {
		return err
	}

	state = fmt.Sprintf("%sed", state)
	story, err := pvt.AssignAndSetStoryState(*user.PivotalId, id, state)
	if err != nil {
		return err
	}

	r.handler.Send(p, fmt.Sprintf("Story %s - %s %s successfully",
		id, story.Name, state))
	return nil
}

func (r bot) sendAuth(p *robots.Payload, cmd utils.Command) error {
	s, err := db.GetSetting(p.UserName, "PIVOTAL_TOKEN")
	if err != nil {
		return err
	}

	if s != nil {
		r.handler.Send(p, "You are already connected with Pivotal.")
		return nil
	}

	msg := `*Authenticating with Pivotal Tracker*
1. Visit your profile here <https://www.pivotaltracker.com/profile>
2. Copy your API token at the bottom of the page
3. Run the command:
   ` + "`/store set PIVOTAL_TOKEN=<token>`"
	r.handler.Send(p, msg)
	return nil
}

func conn(user string) (*pivotal.Pivotal, error) {
	token, err := db.GetSetting(user, "PIVOTAL_TOKEN")
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.New("No PIVOTAL_TOKEN set for @" + user)
	}
	con := pivotal.NewPivotal(token.Value, false)
	return con, nil
}

func projectTable(ps []pivotal.Project) string {
	s := ""

	for _, p := range ps {
		s += fmt.Sprintf("%d - %s\n", p.Id, p.Name)
	}

	return s
}

func (r bot) Description() (description string) {
	return "Pivotal bot\n\tUsage: !pvt <command>\n"
}
