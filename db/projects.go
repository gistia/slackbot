package db

import (
	"database/sql"
	"strconv"
)

type Project struct {
	Id               int    `json:"id"`
	Name             string `json:"name"`
	Channel          string `json:"channel"`
	PivotalId        int64  `json:"pivotal_id"`
	MavenlinkId      int64  `json:"mavenlink_id"`
	MvnSprintStoryId string `json:"sprint_story_id"`
	CreatedBy        string `json:"created_by"`
}

func (p Project) StrPivotalId() string {
	return strconv.FormatInt(p.PivotalId, 10)
}

func (p Project) StrMavenlinkId() string {
	return strconv.FormatInt(p.MavenlinkId, 10)
}

func CreateProject(p Project) error {
	con, err := connect()
	if err != nil {
		return err
	}
	defer con.Close()

	_, err = con.Query(`
    INSERT INTO projects
    (name, pivotal_id, mavenlink_id, created_by,
     mvn_sprint_story_id, channel)
    VALUES ($1, $2, $3, $4, $5, $6)`,
		p.Name, p.PivotalId, p.MavenlinkId, p.CreatedBy,
		p.MvnSprintStoryId, p.Channel)
	return err
}

func GetProjects() ([]Project, error) {
	con, err := connect()
	if err != nil {
		return nil, err
	}
	defer con.Close()

	rows, err := con.Query(`
    SELECT
      "id", "name", "pivotal_id", "mavenlink_id", "created_by",
      "mvn_sprint_story_id", "channel"
    FROM projects WHERE active = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ps := []Project{}
	for rows.Next() {
		pr, err := setProject(rows)
		if err != nil {
			return nil, err
		}

		ps = append(ps, *pr)
	}

	return ps, nil
}

func GetProjectByName(name string) (*Project, error) {
	return GetProjectBy("name", name)
}

func GetProjectByChannel(channel string) (*Project, error) {
	return GetProjectBy("channel", channel)
}

func GetProjectBy(field string, s string) (*Project, error) {
	con, err := connect()
	if err != nil {
		return nil, err
	}
	defer con.Close()

	rows, err := con.Query(`
    SELECT
      "id", "name", "pivotal_id", "mavenlink_id", "created_by",
      "mvn_sprint_story_id", "channel"
    FROM projects
    WHERE "`+field+`" = $1`, s)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return setProject(rows)
}

func GetProject(id int64) (*Project, error) {
	con, err := connect()
	if err != nil {
		return nil, err
	}
	defer con.Close()

	rows, err := con.Query(`
    SELECT
      "id", "name", "pivotal_id", "mavenlink_id", "created_by",
      "mvn_sprint_story_id", "channel"
    FROM projects
    WHERE "id" = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return setProject(rows)
}

func UpdateProject(p Project) error {
	con, err := connect()
	if err != nil {
		return err
	}
	defer con.Close()

	_, err = con.Query(`
    UPDATE
      projects
    SET
      "name" = $1,
      "pivotal_id" = $2,
      "mavenlink_id" = $3,
      "created_by" = $4,
      "mvn_sprint_story_id" = $5,
      "channel" = $6
    WHERE
      "id" = $7`,
		p.Name, p.PivotalId, p.MavenlinkId,
		p.CreatedBy, p.MvnSprintStoryId, p.Channel,
		p.Id)
	return err
}

func setProject(rows *sql.Rows) (*Project, error) {
	p := Project{}
	var mvnSprintStoryId sql.NullString
	var channel sql.NullString
	err := rows.Scan(
		&p.Id, &p.Name, &p.PivotalId, &p.MavenlinkId, &p.CreatedBy,
		&mvnSprintStoryId, &channel)
	if err != nil {
		return nil, err
	}

	if mvnSprintStoryId.Valid {
		p.MvnSprintStoryId = mvnSprintStoryId.String
	}

	if channel.Valid {
		p.Channel = channel.String
	}

	return &p, nil
}
