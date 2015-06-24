package mavenlink

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gistia/slackbot/models"
)

type Mavenlink struct {
	Token   string
	Verbose bool
}

func NewMavenlink(token string, verbose bool) *Mavenlink {
	return &Mavenlink{Token: token, Verbose: verbose}
}

func (mvn *Mavenlink) makeUrl(uri string) string {
	return fmt.Sprintf("https://api.mavenlink.com/api/v1/%s.json", uri)
}

func (mvn *Mavenlink) request(method string, url string, data url.Values) ([]byte, error) {
	var dataIn io.Reader

	if data == nil {
		dataIn = nil
	} else {
		dataIn = bytes.NewBufferString(data.Encode())
	}

	req, err := http.NewRequest(method, url, dataIn)
	if err != nil {
		return nil, err
	}

	auth := fmt.Sprintf("Bearer %s", mvn.Token)
	req.Header.Add("Authorization", auth)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	// fmt.Println("Body:", string(body))

	return body, err
}

func (mvn *Mavenlink) getBody(uri string, filters []string) ([]byte, error) {
	url := mvn.makeUrl(uri)

	if filters != nil {
		url = url + "?"
		for i, f := range filters {
			if i > 0 {
				url = url + "&"
			}
			url = url + f
		}
	}

	if mvn.Verbose {
		fmt.Printf("Requesting: %s...\n", url)
	}

	return mvn.request("GET", url, nil)
}

func (mvn *Mavenlink) get(uri string, filters []string) (*models.Response, error) {
	if filters == nil {
		filters = []string{}
	}

	filters = append(filters, "per_page=200")

	json, err := mvn.getBody(uri, filters)
	if err != nil {
		return nil, err
	}

	resp, err := models.NewFromJson(json)
	return resp, err
}

func (mvn *Mavenlink) post(uri string, params map[string]string) (*models.Response, error) {
	postParams := url.Values{}
	for k, v := range params {
		postParams.Add(k, v)
	}

	json, err := mvn.request("POST", mvn.makeUrl(uri), postParams)
	if err != nil {
		return nil, err
	}

	resp, err := models.NewFromJson(json)
	return resp, err
}

func (mvn *Mavenlink) Projects() ([]models.Project, error) {
	var projects []models.Project
	resp, err := mvn.get("workspaces", nil)

	if err != nil {
		return nil, err
	}

	for k, _ := range resp.Projects {
		p := resp.Projects[k]
		projects = append(projects, p)
	}

	return projects, nil
}

func (mvn *Mavenlink) Project(id string) (*models.Project, error) {
	req := fmt.Sprintf("workspaces/%s", id)
	r, err := mvn.get(req, nil)

	if err != nil {
		return nil, err
	}

	return &r.ProjectList()[0], err
}

func (mvn *Mavenlink) SearchProject(term string) ([]models.Project, error) {
	search := fmt.Sprintf("matching=%s", term)
	r, err := mvn.get("workspaces", []string{search})

	if err != nil {
		return nil, err
	}

	return r.ProjectList(), err
}

func (mvn *Mavenlink) Story(id string) (*models.Story, error) {
	req := fmt.Sprintf("stories/%s", id)
	r, err := mvn.get(req, nil)

	if err != nil {
		return nil, err
	}

	return &r.StoryList()[0], err
}

func (mvn *Mavenlink) Stories(projectId string) ([]models.Story, error) {
	filters := []string{
		fmt.Sprintf("workspace_id=%s", projectId),
		"parents_only=true",
	}
	resp, err := mvn.get("stories", filters)

	if err != nil {
		return nil, err
	}

	return resp.StoryList(), nil
}

func (mvn *Mavenlink) ChildStories(parentId string) ([]models.Story, error) {
	filters := []string{
		fmt.Sprintf("with_parent_id=%s", parentId),
	}
	resp, err := mvn.get("stories", filters)

	if err != nil {
		return nil, err
	}

	return resp.StoryList(), nil
}

func (mvn *Mavenlink) CreateProject(p models.Project) (*models.Project, error) {
	params := map[string]string{"workspace[title]": p.Title}
	_, err := mvn.post("workspaces", params)
	if err != nil {
		return nil, err
	}
	return nil, err
}

func (mvn *Mavenlink) CreateStory(s models.Story) (*models.Story, error) {
	params, err := s.ToParams()
	if err != nil {
		return nil, err
	}

	resp, err := mvn.post("stories", params)
	if err != nil {
		return nil, err
	}

	stories := resp.StoryList()
	if len(stories) > 0 {
		return &stories[0], nil
	}

	return nil, nil
}