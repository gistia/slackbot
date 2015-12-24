# DevBot

## Stories

### Onboard a new user
  - Add to Slack (manual?)
  - Add to GitHub organization
  - Add to Freshbooks
  - Add to Mavenlink
  - Add to Pivotal Tracker

### Add a new project
  - Create a chat room
  - Create GitHub repo(s)
    - Add Slack hooks for the repo (commits, etc.)
  - Create a Freshbook project
  - Create a Mavenlink project
  - Create a Basecamp project
    - Add Slack hooks for the project (new message, files, etc.)
  - Create a Pivotal Tracker project
    - Add PT hooks for the project (?)

### Add user to project
  - Add to Slack chat room
  - Add to GitHub repo(s)
  - Add to Mavenlink project
  - Add to Freshbook project
  - Add to Pivotal Tracker project
  - Add to Basecamp project

### Add a new sprint
  - Create a new GitHub branch
  - Create a new Mavenlink sprint
  - Create (if possible) a Pivotal Tracker epic

### Add tasks to a sprint
  - Create a new task in Mavenlink
  - Create a new task in Pivotal Tracker

### Start a task
  - Mark as started on Pivotal Tracker
  - Mark user as owner on Pivotal Tracker

### Finish a task
  - Mark as finished on Pivotal Tracker
  - Mark as finished on Mavenlink
  - Add hours to Mavenlink
  - Add hours to Freshbooks (optional)

## Project

### Entities

- Slack (slack)
- Slack Channel (channel)
- Slack Users (users)
- Project Management Software (PM software)
- Project
- Tasks / Stories

### Relationships

- Slack have **many** PM software connections
- Each channel have **many** users
- Each channel have **many** projects
- Each project have **many** users
- Each user have **one** PM software user (linked)

### Project level

- [ ] Create a new project
  - [ ] Hourly based vs. Sprint based
- [ ] Connect project to a channel
- [ ] Connect project to project management software (adapters?)
  - [ ] Connect slack users to each PM software
- [ ] Link channel members to PM software
- [ ] Add project member
  - [ ] Cascade to projects in PM software
- [ ] Remove project member
  - [ ] Cascade to projects in PM software
- [ ] Create a new sprint

### Task level

- [ ] Create a new task
- [ ] Start a task
- [ ] Add time to a started task
- [ ] Finish a task (adding time?)
- [ ] Deliver a task
- [ ] Accept a task

## Poker Planning

## GitHub

## Mavenlink

## Pivotal

## Storage

## User

## Time off
