package pivotal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gistia/slackbot/db"
	"github.com/gistia/slackbot/utils"
)

type Pivotal struct {
	Token   string
	Verbose bool
}

type Request struct {
	Type                 string
	Method               string
	Uri                  string
	Data                 map[string]string
	Token                string
	Story                *Story
	Project              *Project
	ProjectMembership    *ProjectMembership
	NewProjectMembership *NewProjectMembership
}

type Response struct {
	Projects           []Project           `json:"projects"`
	Stories            []Story             `json:"stories"`
	ProjectMemberships []ProjectMembership `json:"project_memberships"`
	Project            Project             `json:"project"`
	ProjectMembership  ProjectMembership   `json:"project_membership"`
	Story              Story               `json:"story"`
	Error              Error               `json:"error"`
}

type Project struct {
	Id              int64  `json:"id,omitempty"`
	Kind            string `json:"kind,omitempty"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Version         int64  `json:"version,omitempty"`
	IterationLength int    `json:"iteration_length,omitempty"`
}

type Story struct {
	Id          int64  `json:"id,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Estimate    int    `json:"estimate,omitempty"`
	State       string `json:"current_state,omitempty"`
	Url         string `json:"url,omitempty"`
	Type        string `json:"story_type,omitempty"`
	ProjectId   int64  `json:"project_id,omitempty"`
}

type ProjectMembership struct {
	Id           int64  `json:"id,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Person       Person `json:"person"`
	ProjectId    int64  `json:"project_id"`
	Role         string `json:"role"`
	ProjectColor string `json:"project_color"`
	LastViewedAt string `json:"last_viewed_at"`
}

type Person struct {
	Id       int64  `json:"id,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Initials string `json:"initials"`
	Username string `json:"username"`
}

type NewProjectMembership struct {
	PersonId     int64  `json:"person_id"`
	ProjectColor string `json:"project_color,omitempty"`
	Role         string `json:"role"`
}

type Error struct {
	Code           string `json:"code"`
	Kind           string `json:"kind"`
	Error          string `json:"error"`
	PossibleFix    string `json:"possible_fix"`
	GeneralProject string `json:"general_problem"`
}

func NewPivotal(token string, verbose bool) *Pivotal {
	return &Pivotal{Token: token, Verbose: verbose}
}

func NewFor(user string) (*Pivotal, error) {
	token, err := db.GetSetting(user, "PIVOTAL_TOKEN")
	if err != nil {
		return nil, err
	}
	return NewPivotal(token.Value, false), nil
}

func (r *Request) request(method string, uri string, data url.Values) ([]byte, error) {
	url := fmt.Sprintf("https://www.pivotaltracker.com/services/v5/%s", uri)

	fmt.Println("URL:", url)

	return utils.Request(method, url, data,
		map[string]string{"X-TrackerToken": r.Token})
}

func (r *Request) Send() (*Response, error) {
	uri := r.Uri
	// fmt.Printf("Type: %s Uri: %s\n", r.Type, r.Uri)
	if uri == "" {
		uri = r.Type
	}

	var payload []byte
	var err error

	if r.Story != nil {
		data, err := json.Marshal(r.Story)
		if err != nil {
			return nil, err
		}
		url := fmt.Sprintf("https://www.pivotaltracker.com/services/v5/%s", uri)
		headers := map[string]string{
			"X-TrackerToken": r.Token,
			"Content-Type":   "application/json",
		}
		fmt.Println("Request:", url)
		fmt.Println("Request payload:", string(data))
		payload, err = utils.RequestRaw(
			r.Method, url, bytes.NewBuffer(data), headers)
	} else if r.NewProjectMembership != nil {
		data, err := json.Marshal(r.NewProjectMembership)
		if err != nil {
			return nil, err
		}
		url := fmt.Sprintf("https://www.pivotaltracker.com/services/v5/%s", uri)
		headers := map[string]string{
			"X-TrackerToken": r.Token,
			"Content-Type":   "application/json",
		}
		fmt.Println("Request:", url)
		fmt.Println("Request payload:", string(data))
		payload, err = utils.RequestRaw(
			r.Method, url, bytes.NewBuffer(data), headers)
	} else if r.Project != nil {
		data, err := json.Marshal(r.Project)
		if err != nil {
			return nil, err
		}
		url := fmt.Sprintf("https://www.pivotaltracker.com/services/v5/%s", uri)
		headers := map[string]string{
			"X-TrackerToken": r.Token,
			"Content-Type":   "application/json",
		}
		fmt.Println("Request:", url)
		fmt.Println("Request payload:", string(data))
		payload, err = utils.RequestRaw(
			r.Method, url, bytes.NewBuffer(data), headers)
	} else {
		values := url.Values{}
		for k, v := range r.Data {
			values.Add(k, v)
		}
		payload, err = r.request(r.Method, uri, values)
	}

	if err != nil {
		return nil, err
	}

	fmt.Println("Payload:", string(payload))
	wrapped := string(payload)

	if strings.Contains(wrapped, "\"kind\":\"error\"") {
		wrapped = fmt.Sprintf("{\"error\":%s}", wrapped)
		resp, err := NewFromJson([]byte(wrapped))
		if err != nil {
			return nil, err
		}
		pvtError := resp.Error
		msg := fmt.Sprintf("%s - %s\n", pvtError.Code, pvtError.Error)
		if pvtError.GeneralProject != "" {
			msg += fmt.Sprintf("Details: %s\n", pvtError.GeneralProject)
		}
		if pvtError.PossibleFix != "" {
			msg += fmt.Sprintf("Possible fix: %s\n", pvtError.PossibleFix)
		}
		return nil, errors.New(msg)
	}

	if wrapped != "" {
		wrapped = fmt.Sprintf("{\"%s\":%s}", r.Type, wrapped)
	}
	fmt.Println("Wrapped:", string(wrapped))
	resp, err := NewFromJson([]byte(wrapped))
	return resp, err
}

func NewFromJson(jsonData []byte) (*Response, error) {
	var b *Response

	err := json.Unmarshal(jsonData, &b)

	// if len(b.Errors) > 0 {
	// 	msg := ""
	// 	for _, e := range b.Errors {
	// 		msg += fmt.Sprintf("%s (%s)\n", e.Message, e.Type)
	// 	}
	// 	return nil, errors.New(msg)
	// }

	return b, err
}

func (pvt *Pivotal) Projects() ([]Project, error) {
	req := Request{Token: pvt.Token, Type: "projects", Method: "GET"}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}

	return r.Projects, nil
}

func (pvt *Pivotal) GetProject(id string) (*Project, error) {
	uri := fmt.Sprintf("projects/%s", id)
	req := Request{
		Token:  pvt.Token,
		Type:   "project",
		Method: "GET",
		Uri:    uri,
	}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}

	return &r.Project, nil
}

func (pvt *Pivotal) Stories(p string) ([]Story, error) {
	uri := fmt.Sprintf("projects/%s/stories", p)
	req := Request{Token: pvt.Token, Type: "stories", Method: "GET", Uri: uri}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}

	return r.Stories, nil
}

func (pvt *Pivotal) SetStoryState(id string, state string) (*Story, error) {
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}

	story := Story{Id: nid, State: state}
	return pvt.UpdateStory(story)
}

func (pvt *Pivotal) UpdateStory(story Story) (*Story, error) {
	req := Request{
		Token:  pvt.Token,
		Type:   "story",
		Method: "PUT",
		Uri:    fmt.Sprintf("stories/%d", story.Id),
		Story:  &story,
	}

	story.Url = ""

	r, err := req.Send()
	if err != nil {
		return nil, err
	}
	return &r.Story, nil
}

func (pvt *Pivotal) CreateProject(project Project) (*Project, error) {
	fmt.Println("Project", project)
	req := Request{
		Token:   pvt.Token,
		Type:    "project",
		Method:  "POST",
		Uri:     "projects",
		Project: &project,
	}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}
	fmt.Println("Project:", r.Project)
	return &r.Project, nil
}

func (pvt *Pivotal) UpdateProject(project Project) (*Project, error) {
	req := Request{
		Token:   pvt.Token,
		Type:    "project",
		Method:  "PUT",
		Uri:     fmt.Sprintf("projects/%d", project.Id),
		Project: &project,
	}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}
	return &r.Project, nil
}

func (pvt *Pivotal) CreateStory(story Story) (*Story, error) {
	req := Request{
		Token:  pvt.Token,
		Type:   "story",
		Method: "POST",
		Uri:    fmt.Sprintf("projects/%d/stories", story.ProjectId),
		Story:  &story,
	}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}
	return &r.Story, nil
}

func (pvt *Pivotal) CreateProjectMembership(projectId string, personId int64, role string) (*ProjectMembership, error) {
	req := Request{
		Token:                pvt.Token,
		Type:                 "project_membership",
		Method:               "POST",
		Uri:                  fmt.Sprintf("projects/%s/memberships", projectId),
		NewProjectMembership: &NewProjectMembership{PersonId: personId, Role: role},
	}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}
	return &r.ProjectMembership, nil
}

func (pvt *Pivotal) GetProjectMemberships(projectId string) ([]ProjectMembership, error) {
	uri := fmt.Sprintf("projects/%s/memberships", projectId)
	req := Request{
		Token:  pvt.Token,
		Type:   "project_memberships",
		Method: "GET",
		Uri:    uri,
	}

	r, err := req.Send()
	if err != nil {
		return nil, err
	}

	return r.ProjectMemberships, nil
}
