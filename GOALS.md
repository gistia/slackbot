# DevBot

## Project

### Entities

- Slack (slack)
- Slack Channel (channel)
- Slack Users (users)
- Project Management Software (PM software)
- Project
- Tasks

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
