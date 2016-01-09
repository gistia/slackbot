package robots

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gistia/slackbot/db"
	"github.com/gistia/slackbot/mavenlink"
	"github.com/gistia/slackbot/pivotal"
	"github.com/gistia/slackbot/robots"
	"github.com/gistia/slackbot/utils"
)

type bot struct {
	handler utils.SlackHandler
}

func init() {
	handler := utils.NewSlackHandler("Project", ":books:")
	s := &bot{handler: handler}
	robots.RegisterRobot("project", s)
	robots.RegisterRobot("pr", s)
}

func (r bot) Run(p *robots.Payload) string {
	go r.DeferredAction(p)
	return ""
}

func (r bot) DeferredAction(p *robots.Payload) {
	ch := utils.NewCmdHandler(p, r.handler, "project")
	ch.Handle("list", r.list)
	ch.Handle("link", r.link)
	ch.Handle("stories", r.stories)
	ch.Handle("mystories", r.myStories)
	ch.Handle("setsprint", r.setSprint)
	ch.Handle("addsprint", r.addSprint)
	ch.Handle("setchannel", r.setChannel)
	ch.Handle("addstory", r.addStory)
	ch.Handle("addstories", r.addStories)
	ch.Handle("start", r.startTask)
	ch.Handle("finish", r.finishTask)
	ch.Handle("deliver", r.deliverTask)
	ch.Handle("create", r.create)
	ch.Handle("rename", r.rename)
	ch.Handle("members", r.members)
	ch.Handle("addmember", r.addmember)
	ch.Handle("unassigned", r.unassigned)
	ch.Handle("assign", r.assign)
	ch.Handle("estimate", r.estimate)
	ch.Handle("addtime", r.addTime)
	ch.Handle("setbudget", r.setBudget)
	ch.Handle("current", r.getCurrent)
	ch.Handle("clearcurrent", r.clearCurrent)
	ch.HandleDefault(r.list)
	ch.Process(p.Text)
}

func (r bot) setBudget(p *robots.Payload, cmd utils.Command) error {
	args, err := cmd.ParseArgs("project", "budget")
	if err != nil {
		return err
	}

	name, budgetStr := args[0], args[1]

	pr, err := db.GetProjectByName(name)
	if err != nil {
		return err
	}

	if pr == nil {
		r.handler.Send(p, "Project *"+name+"* not found")
		return nil
	}

	mvn, err := mavenlink.NewFor(p.UserName)
	if err != nil {
		return err
	}

	mvnProj, err := mvn.GetProject(pr.StrMavenlinkId())
	if err != nil {
		return err
	}

	budget, err := strconv.ParseFloat(budgetStr, 64)
	if err != nil {
		return err
	}

	mvnProj, err = mvn.UpdateProjectBudget(mvnProj, budget)
	if err != nil {
		return err
	}

	r.handler.Send(p, fmt.Sprintf("Set budget of *%s - %s* to *%.2f*",
		mvnProj.Id, mvnProj.Title, mvnProj.GetBudget()))
	return nil
}

func (r bot) addTime(p *robots.Payload, cmd utils.Command) error {
	args, err := cmd.ParseArgs("mavenlink-id", "time-in-hours")
	if err != nil {
		return err
	}

	mvnID, timeStr := args[0], args[1]
	hours, err := strconv.ParseFloat(timeStr, 64)
	if err != nil {
		return err
	}

	mvn, err := mavenlink.NewFor(p.UserName)
	if err != nil {
		return err
	}

	story, err := mvn.GetStory(mvnID)
	if err != nil {
		return err
	}

	minutes := int(hours * 60)
	_, err = mvn.AddTimeEntry(story, minutes)
	if err != nil {
		return err
	}

	r.handler.Send(p, fmt.Sprintf("Added *%.1f* hours to story *%s - %s*",
		hours, story.Id, story.Title))
	return nil
}

func (r bot) estimate(p *robots.Payload, cmd utils.Command) error {
	args, err := cmd.ParseArgs("pivotal-id", "estimate")
	if err != nil {
		return err
	}

	pvtId, estimate := args[0], args[1]

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	// pvtStory, err := pvt.GetStory(pvtId)
	// if err != nil {
	// 	return err
	// }
	//
	numEstimate, err := strconv.Atoi(estimate)
	if err != nil {
		return err
	}

	story, err := pvt.EstimateStory(pvtId, numEstimate)
	if err != nil {
		return err
	}

	s := "Story successfully updated:\n"
	r.handler.SendWithAttachments(p, s, []robots.Attachment{
		utils.FmtAttachment("", story.Name, story.Url, ""),
	})

	return nil
}

func (r bot) myStories(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	var pr *db.Project
	var err error

	if name == "" {
		pr, err = db.GetProjectByChannel(p.ChannelName)
	} else {
		pr, err = db.GetProjectByName(name)
	}

	if err != nil {
		return err
	}

	if pr == nil {
		r.handler.Send(p, "Missing project name.")
		return nil
	}

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	user, err := db.GetUserByName(p.UserName)
	if err != nil {
		return err
	}

	filter := map[string]string{
		"owned_by": user.StrPivotalId(),
		"state":    "started,finished",
	}
	stories, err := pvt.FilteredStories(pr.StrPivotalId(), filter)
	if err != nil {
		return err
	}

	if len(stories) < 1 {
		r.handler.Send(p, "No open stories in project *"+pr.Name+"* for *"+p.UserName+"*")
		return nil
	}

	str := "Current stories in project *" + pr.Name + "* for *" + p.UserName + "*:\n"
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

func (r bot) unassigned(p *robots.Payload, cmd utils.Command) error {
	res, err := cmd.ParseArgs("project")
	if err != nil {
		return err
	}

	name := res[0]
	pr, err := getProject(name)
	if err != nil {
		return err
	}

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	msg := "Unassigned stories for *" + name + "*:\n"

	stories, err := pvt.GetUnassignedStories(pr.StrPivotalId())

	if len(stories) < 1 {
		r.handler.Send(p, "No unassigned stories for *"+name+"*")
		return nil
	}

	for _, s := range stories {
		msg += fmt.Sprintf("%d - %s\n", s.Id, s.Name)
	}

	r.handler.Send(p, msg)
	return nil
}

func (r bot) assign(p *robots.Payload, cmd utils.Command) error {
	res, err := cmd.ParseArgs("pivotal-story-id", "username")
	if err != nil {
		return err
	}

	storyId, username := res[0], res[1]

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	user, err := db.GetUserByName(username)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("User *" + username + "* not found")
	}

	story, err := pvt.AssignStory(storyId, *user.PivotalId)
	if err != nil {
		return err
	}

	s := "Story successfully updated:\n"
	r.handler.SendWithAttachments(p, s, []robots.Attachment{
		utils.FmtAttachment("", story.Name, story.Url, ""),
	})

	return nil
}

func (r bot) addmember(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		err := errors.New(
			"Missing project name. Use `!project addmember <project> <username>`")
		return err
	}
	username := cmd.Arg(1)
	if name == "" {
		err := errors.New(
			"Missing user name. Use `!project addmember <project> <username>`")
		return err
	}

	pr, err := getProject(name)
	if err != nil {
		return err
	}

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	if pr == nil {
		r.handler.Send(p, "Project *"+name+"* not found")
		return nil
	}

	user, err := db.GetUserByName(username)
	if pr == nil {
		r.handler.Send(p, "Project *"+name+"* not found")
		return nil
	}
	if user == nil {
		r.handler.Send(p, "User *"+username+"* not found")
		return nil
	}

	_, err = pvt.CreateProjectMembership(pr.StrPivotalId(), *user.PivotalId, "member")
	if err != nil {
		return err
	}

	r.handler.Send(p, "New member *"+username+"* added to *"+name+"*")
	return nil
}

func (r bot) members(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		err := errors.New(
			"Missing project name. Use `!project addtask <project> <task-name>`")
		return err
	}
	pr, err := getProject(name)
	if err != nil {
		return err
	}

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	if pr == nil {
		r.handler.Send(p, "Project *"+name+"* not found")
		return nil
	}

	members, err := pvt.GetProjectMemberships(pr.StrPivotalId())
	if err != nil {
		return err
	}

	s := "Pivotal members for project *" + pr.Name + "*:\n"
	for _, m := range members {
		s += fmt.Sprintf("%d - %s\n", m.Person.Id, m.Person.Name)
	}

	r.handler.Send(p, s)
	return nil
}

func (r bot) rename(p *robots.Payload, cmd utils.Command) error {
	old := cmd.Arg(0)
	new := cmd.Arg(1)
	if new == "" || old == "" {
		r.handler.Send(p, "You need to provide the old and new name. Usage: `!project rename <old-name> <new-name>`")
		return nil
	}

	pr, err := db.GetProjectByName(old)
	if err != nil {
		return err
	}

	if pr == nil {
		r.handler.Send(p, "Project *"+old+"* not found.")
		return nil
	}

	pr.Name = new
	err = db.UpdateProject(*pr)
	if err != nil {
		return err
	}

	r.handler.Send(p, "Project *"+old+"* renamed to *"+new+"*")
	return nil
}

func (r bot) create(p *robots.Payload, cmd utils.Command) error {
	alias := cmd.Arg(0)
	if alias == "" {
		r.handler.Send(p, "Missing project alias. Usage: `!project createproject <alias> <long-name>`")
		return nil
	}
	name := cmd.StrFrom(1)
	if name == "" {
		r.handler.Send(p, "Missing project name. Usage: `!project createproject <alias> <long-name>`")
		return nil
	}
	mvn, err := mavenlink.NewFor(p.UserName)
	if err != nil {
		return err
	}
	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}
	pvtProject := pivotal.Project{
		Name: name,
		// PointScale: "1,2,3,4,5,6,7,8,9,10,16,20",
	}
	pvtNewProject, err := pvt.CreateProject(pvtProject)
	if err != nil {
		return err
	}
	mvnProject := mavenlink.Project{
		Title:       name,
		Description: fmt.Sprintf("[pvt:%s]", pvtNewProject.Id),
		CreatorRole: "maven",
	}
	mvnNewProject, err := mvn.CreateProject(mvnProject)
	if err != nil {
		return err
	}
	if mvnNewProject == nil {
		return errors.New("Mavenlink returned a nil project")
	}
	pvtNewProject.Description = "[mvn:" + mvnNewProject.Id + "]"
	pvtNewProject, err = pvt.UpdateProject(*pvtNewProject)
	if err != nil {
		return err
	}

	err = r.makeLink(p, alias, mvnNewProject.Id, strconv.FormatInt(pvtNewProject.Id, 10))
	if err != nil {
		return err
	}

	r.handler.Send(p, "Project *"+name+"* created on Pivotal and Mavenlink.")
	return nil
}

func (r bot) updateCurrentTask(p *robots.Payload, storyID, status string) error {
	if storyID == "" {
		assignment, err := db.GetAssignment(p.UserName, "CurrentStory")
		if err != nil {
			return err
		}

		if assignment == nil || assignment.Value == "" {
			r.handler.DirectSend(p, "you don't have any current stories. To start a new story use `!project start <pivotal-id>`")
			return nil
		}

		storyID = assignment.Value
	}

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	story, err := pvt.GetStory(storyID)
	if err != nil {
		return err
	}

	story, err = pvt.SetStoryState(story.GetStringId(), status)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Task *%d - %s* is now *%s*", story.Id, story.Name, status)
	r.handler.DirectSend(p, msg)

	return nil
}

func (r bot) finishTask(p *robots.Payload, cmd utils.Command) error {
	return r.updateCurrentTask(p, cmd.Arg(0), "finished")
}

func (r bot) deliverTask(p *robots.Payload, cmd utils.Command) error {
	err := r.updateCurrentTask(p, cmd.Arg(0), "delivered")
	if err != nil {
		return err
	}

	err = db.ClearAssignment(p.UserName, "CurrentStory")
	if err != nil {
		return nil
	}

	r.handler.DirectSend(p, "you have no current story. To start the next one use `!project start <pivotal-id>`")
	return nil
}

func (r bot) clearCurrent(p *robots.Payload, cmd utils.Command) error {
	err := db.ClearAssignment(p.UserName, "CurrentStory")
	if err != nil {
		return nil
	}

	r.handler.DirectSend(p, "current story cleared. To start the next one use `!project start <pivotal-id>`")
	return nil
}

func (r bot) getCurrent(p *robots.Payload, cmd utils.Command) error {
	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	storyID := cmd.Arg(0)
	if storyID != "" {
		story, err := pvt.GetStory(storyID)
		if err != nil {
			return err
		}

		if story == nil {
			return errors.New("Story with id *" + storyID + "* not found.")
		}

		_, err = db.SetAssignment(p.UserName, "CurrentStory", storyID)
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("your current story is now *%d - %s* [*%s*]", story.Id, story.Name, story.State)
		r.handler.DirectSend(p, msg)
	}

	assignment, err := db.GetAssignment(p.UserName, "CurrentStory")
	if err != nil {
		return err
	}

	fmt.Println(assignment)

	if assignment == nil || assignment.Value == "" {
		r.handler.DirectSend(p, "you don't have any current stories. To start a new story use `!project start <pivotal-id>`")
		return nil
	}

	storyID = assignment.Value

	story, err := pvt.GetStory(storyID)
	if err != nil {
		return err
	}

	s := fmt.Sprintf("@%s current story [*%s*]:\n", p.UserName, story.State)
	r.handler.SendWithAttachments(p, s, []robots.Attachment{
		utils.FmtAttachment("", story.Name, story.Url, ""),
	})

	return nil
}

func (r bot) startTask(p *robots.Payload, cmd utils.Command) error {
	storyID := cmd.Arg(0)
	if storyID == "" {
		return errors.New("Missing pivotal-story-id.")
	}

	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return err
	}

	pvtStory, err := pvt.GetStory(storyID)
	if err != nil {
		return err
	}

	pvtStory, err = pvt.SetStoryState(pvtStory.GetStringId(), "started")
	if err != nil {
		return err
	}

	_, err = db.SetAssignment(p.UserName, "CurrentStory", strconv.FormatInt(pvtStory.Id, 10))
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Task %d - %s started and marked is your current story",
		pvtStory.Id, pvtStory.Name)
	r.handler.DirectSend(p, msg)

	return nil
}

func getProject(name string) (*db.Project, error) {
	pr, err := db.GetProjectByName(name)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		err := errors.New("Project *" + name + "* not found.")
		return nil, err
	}

	return pr, nil
}

func getChannelProject(p *robots.Payload) (*db.Project, error) {
	pr, err := db.GetProjectByChannel(p.ChannelName)
	if err != nil {
		return nil, err
	}
	return pr, err
}

func (r bot) addStories(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		pr, err := getChannelProject(p)
		if err != nil {
			return err
		}
		if pr == nil {
			err := errors.New(
				"No project assigned to this channel. Use `!project addstories <project>`")
			return err
		}
		name = pr.Name
	}
	if name == "" {
		err := errors.New(
			"Missing project name. Use `!project addstories <project>`")
		return err
	}
	appUrl := os.Getenv("APP_URL")
	url := appUrl + "/addstories?channel=" + p.ChannelName +
		"&channel_id=" + p.ChannelID + "&user=" + p.UserName +
		"&project=" + name
	a := utils.FmtAttachment("", "Click here to add stories", url, "Add stories to this project by following the link")
	r.handler.SendWithAttachments(p, "", []robots.Attachment{a})
	return nil
}

func CreateStory(userName string, projectName string, storyName string, storyType string) (*pivotal.Story, *mavenlink.Story, error) {
	pr, err := getProject(projectName)
	if err != nil {
		return nil, nil, err
	}
	mvn, err := mavenlink.NewFor(userName)
	if err != nil {
		return nil, nil, err
	}
	pvt, err := pivotal.NewFor(userName)
	if err != nil {
		return nil, nil, err
	}

	pvtStory := pivotal.Story{
		Name:      storyName,
		ProjectId: pr.PivotalId,
		Type:      storyType,
	}
	pvtNewStory, err := pvt.CreateStory(pvtStory)
	if err != nil {
		return nil, nil, err
	}

	desc := fmt.Sprintf("[pvt:%d]", pvtNewStory.Id)
	mvnStory := mavenlink.Story{
		Title:       storyName,
		Description: desc,
		ParentId:    pr.MvnSprintStoryId,
		WorkspaceId: strconv.FormatInt(pr.MavenlinkId, 10),
		StoryType:   "task",
	}
	mvnNewStory, err := mvn.CreateStory(mvnStory)
	if err != nil {
		return nil, nil, err
	}

	tmpStory := pivotal.Story{
		Id:          pvtNewStory.Id,
		Description: "[mvn:" + mvnNewStory.Id + "]",
	}
	pvtNewStory, err = pvt.UpdateStory(tmpStory)
	if err != nil {
		return nil, nil, err
	}

	return pvtNewStory, mvnNewStory, err
}

func (r bot) addStory(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		err := errors.New(
			"Missing project name. Use `!project addtask <project> [type:<story-type>] <task-name>`")
		return err
	}

	storyType := cmd.Param("type")
	if storyType == "" {
		storyType = "feature"
	}

	storyName := strings.Join(cmd.ArgsFrom(1), " ")
	if storyName == "" {
		err := errors.New(
			"Missing story name. Use `!project addtask <project> <task-name>`")
		return err
	}

	pvtNewStory, mvnNewStory, err := CreateStory(p.UserName, name, storyName, storyType)
	fmt.Printf("%v+\n", pvtNewStory)
	fmt.Printf("%v+\n", mvnNewStory)
	if err != nil {
		return err
	}

	pvtAtt := utils.FmtAttachment(
		fmt.Sprintf("- Pivotal %d - %s\n", pvtNewStory.Id, pvtNewStory.Name),
		fmt.Sprintf("Pivotal %d - %s\n", pvtNewStory.Id, pvtNewStory.Name),
		pvtNewStory.Url, "")
	mvnAtt := utils.FmtAttachment(
		fmt.Sprintf("- Mavenlink %s - %s\n", mvnNewStory.Id, mvnNewStory.Title),
		fmt.Sprintf("Mavenlink %s - %s\n", mvnNewStory.Id, mvnNewStory.Title),
		mvnNewStory.URL(), "")

	s := "Stories successfully added:\n"
	r.handler.SendWithAttachments(p, s, []robots.Attachment{pvtAtt, mvnAtt})
	return nil
}

func (r bot) addSprint(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		r.handler.Send(p, "Missing project name")
		return nil
	}

	ps, err := db.GetProjectByName(name)
	if err != nil {
		return err
	}

	mvn, err := mavenlink.NewFor(p.UserName)
	if err != nil {
		return err
	}

	sprintName := cmd.StrFrom(1)
	if sprintName == "" {
		sprintName = "Sprint 1"

		if ps.MvnSprintStoryId != "" {
			s, err := mvn.GetStory(ps.MvnSprintStoryId)
			if err != nil {
				return err
			}

			matched, err := regexp.MatchString(`Sprint [\d]+`, s.Title)
			if err != nil {
				return err
			}
			if matched {
				num, err := strconv.ParseInt(strings.Split(s.Title, " ")[1], 10, 64)
				if err != nil {
					return err
				}
				sprintName = fmt.Sprintf("Sprint %d", (num + 1))
			}
		}
	}

	s := mavenlink.Story{
		Title:       sprintName,
		WorkspaceId: strconv.FormatInt(ps.MavenlinkId, 10),
		StoryType:   "milestone",
	}

	ns, err := mvn.CreateStory(s)
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", ns)

	ps.MvnSprintStoryId = ns.Id
	err = db.UpdateProject(*ps)
	if err != nil {
		return err
	}

	s = *ns
	r.handler.Send(p, "Added new sprint to *"+ps.Name+"*: "+s.Id+" - "+s.Title)
	return nil
}

func (r bot) setChannel(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		r.handler.Send(p, "Missing project name")
		return nil
	}

	ps, err := db.GetProjectByName(name)
	if err != nil {
		return err
	}

	ps.Channel = p.ChannelName
	if err := db.UpdateProject(*ps); err != nil {
		return err
	}

	r.handler.Send(p, "Project *"+name+"* assigned to *"+ps.Channel+"* channel.")
	return nil
}

func (r bot) stories(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)

	var ps *db.Project
	var err error
	if name != "" {
		ps, err = db.GetProjectByName(name)
		if err != nil {
			return err
		}
		if ps == nil {
			r.handler.Send(p, "Project *"+name+"* not found")
			return nil
		}
	}

	if ps == nil {
		ps, err = db.GetProjectByChannel(p.ChannelName)
		if err != nil {
			return err
		}
		if ps == nil {
			r.handler.Send(p, "Missing project name")
			return nil
		}
	}

	mvn, err := mavenlink.NewFor(p.UserName)
	if err != nil {
		return err
	}

	sprint, err := mvn.GetStory(ps.MvnSprintStoryId)
	if err != nil {
		return err
	}

	stories, err := mvn.GetChildStories(ps.MvnSprintStoryId)
	if err != nil {
		return err
	}

	r.handler.Send(p, "Mavenlink stories for *"+ps.Name+"*, sprint *"+sprint.Title+"*:")
	atts := mavenlink.FormatStories(stories)
	for _, a := range atts {
		r.handler.SendWithAttachments(p, "", []robots.Attachment{a})
	}

	var totalEstimated int64
	var totalLogged int64
	for _, s := range stories {
		totalEstimated += s.TimeEstimateInMinutes
		totalLogged += s.LoggedBillableTimeInMinutes
	}

	s := ""
	if totalEstimated > 0 {
		s += fmt.Sprintf("Total estimated: %s", utils.FormatHour(totalEstimated))
	}
	if totalLogged > 0 {
		if totalEstimated > 0 {
			s += " - "
		}
		s += fmt.Sprintf("Total logged: %s", utils.FormatHour(totalLogged))
	}
	if s != "" {
		r.handler.Send(p, s)
	}

	return nil
}

func (r bot) setSprint(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		r.handler.Send(p, "Missing project name")
		return nil
	}

	id := cmd.Arg(1)
	if id == "" {
		r.handler.Send(p, "Missing mavenlink story id to assign as current sprint")
		return nil
	}

	ps, err := db.GetProjectByName(name)
	if err != nil {
		return err
	}

	mvn, err := mavenlink.NewFor(p.UserName)
	if err != nil {
		return err
	}

	mvnStory, err := mvn.GetStory(id)
	if err != nil {
		return err
	}

	if mvnStory == nil {
		r.handler.Send(p, "Story with id "+id+" wasn't found")
		return nil
	}

	fmt.Println("Got story", mvnStory.Id)
	ps.MvnSprintStoryId = mvnStory.Id
	if err := db.UpdateProject(*ps); err != nil {
		return err
	}

	r.handler.Send(p, "Project *"+name+"* updated.")
	return nil
}

func (r bot) list(p *robots.Payload, cmd utils.Command) error {
	ps, err := db.GetProjects()
	if err != nil {
		return err
	}

	if ps == nil || len(ps) < 1 {
		r.handler.Send(p, "There are no linked projects currently. Use `link` command to add one.")
		return nil
	}

	s := ""

	for _, pr := range ps {
		fmt.Println(" *** Project", pr)
		fmt.Println("     Starting PVT", strconv.FormatInt(pr.PivotalId, 10))
		pvtPr, err := r.getPvtProject(p, strconv.FormatInt(pr.PivotalId, 10))
		if err != nil {
			return err
		}
		fmt.Println("     Starting MVN", strconv.FormatInt(pr.MavenlinkId, 10))
		mvnPr, err := r.getMvnProject(p, strconv.FormatInt(pr.MavenlinkId, 10))
		if err != nil {
			return err
		}

		sprintInfo := ""
		if pr.MvnSprintStoryId != "" {
			mvn, err := mavenlink.NewFor(p.UserName)
			if err != nil {
				return err
			}

			fmt.Println("     Starting Story", pr.MvnSprintStoryId)
			sprintStory, err := mvn.GetStory(pr.MvnSprintStoryId)
			fmt.Println("     Sotry result", err)
			if err != nil {
				fmt.Println("     Returning error:", err)
				return err
			}

			sprintInfo = "Current sprint: " + sprintStory.Title + "\n"
			fmt.Println("     Finished Story")
		}

		s += fmt.Sprintf(
			"*%s*\nPivotal %d - %s to Mavenlink %s - %s\n%s",
			pr.Name, pvtPr.Id, pvtPr.Name, mvnPr.Id, mvnPr.Title, sprintInfo)
		fmt.Println("     Finished Project")
	}

	fmt.Println("     Sending response:", s)
	r.handler.Send(p, s)
	return nil
}

func (r bot) link(p *robots.Payload, cmd utils.Command) error {
	name := cmd.Arg(0)
	if name == "" {
		r.handler.Send(p, "Missing project name. Usage: !project link name mvn:id pvt:id")
		return nil
	}
	mvn := cmd.Param("mvn")
	if mvn == "" {
		r.handler.Send(p, "Missing mavenlink project. Usage: !project link name mvn:id pvt:id")
		return nil
	}
	pvt := cmd.Param("pvt")
	if pvt == "" {
		r.handler.Send(p, "Missing pivotal project. Usage: !project link name mvn:id pvt:id")
		return nil
	}

	err := r.makeLink(p, name, mvn, pvt)
	if err != nil {
		return err
	}

	return nil
}

func (r bot) getMvnProject(p *robots.Payload, id string) (*mavenlink.Project, error) {
	mvn, err := mavenlink.NewFor(p.UserName)
	if err != nil {
		return nil, err
	}
	return mvn.GetProject(id)
}

func (r bot) getPvtProject(p *robots.Payload, id string) (*pivotal.Project, error) {
	pvt, err := pivotal.NewFor(p.UserName)
	if err != nil {
		return nil, err
	}
	return pvt.GetProject(id)
}

func (r bot) makeLink(p *robots.Payload, name string, mvnId string, pvtId string) error {
	prj, err := db.GetProjectByName(name)
	if err != nil {
		return err
	}

	if prj != nil {
		r.handler.Send(p, fmt.Sprintf("Project with name %s already exists.", name))
		return nil
	}

	mvnProject, err := r.getMvnProject(p, mvnId)
	if err != nil {
		msg := fmt.Sprintf("Error loading mavenlink project %s: %s", mvnId, err.Error())
		return errors.New(msg)
	}

	pvtProject, err := r.getPvtProject(p, pvtId)
	if err != nil {
		msg := fmt.Sprintf("Error loading pivotal project %s: %s", pvtId, err.Error())
		return errors.New(msg)
	}

	mvnInt, err := strconv.ParseInt(mvnProject.Id, 10, 64)
	if err != nil {
		return err
	}
	pvtInt := pvtProject.Id

	project := db.Project{
		Name:        name,
		MavenlinkId: mvnInt,
		PivotalId:   pvtInt,
		CreatedBy:   p.UserName,
	}
	err = db.CreateProject(project)
	if err != nil {
		return err
	}

	r.handler.Send(p, fmt.Sprintf("Project %s linked %s - %s and %d - %s", name,
		mvnProject.Id, mvnProject.Title,
		pvtProject.Id, pvtProject.Name))

	return err
}

func (r bot) sendError(p *robots.Payload, err error) {
	msg := fmt.Sprintf("Error running project command: %s\n", err.Error())
	r.handler.Send(p, msg)
}

func (r bot) Description() (description string) {
	return "Project bot\n\tUsage: !project <command>\n"
}
