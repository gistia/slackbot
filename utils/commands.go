package utils

import (
	"errors"
	"strings"

	"github.com/gistia/slackbot/robots"
)

type HandlerFunc func(*robots.Payload, Command) error

type CmdHandler struct {
	name     string
	payload  *robots.Payload
	handlers map[string]HandlerFunc
	msgr     SlackHandler
}

func NewCmdHandler(p *robots.Payload, h SlackHandler, name string) CmdHandler {
	return CmdHandler{
		name:     name,
		payload:  p,
		handlers: map[string]HandlerFunc{},
		msgr:     h,
	}
}

func (c *CmdHandler) Handle(cmd string, handler HandlerFunc) {
	c.handlers[cmd] = handler
}

func (c *CmdHandler) HandleMany(cmds []string, handler HandlerFunc) {
	for _, cmd := range cmds {
		c.handlers[cmd] = handler
	}
}

func (c *CmdHandler) HandleDefault(handler HandlerFunc) {
	c.handlers["_default"] = handler
}

func (c *CmdHandler) Process(s string) {
	cmd := NewCommand(s)

	if cmd.IsDefault() {
		if h := c.handlers["_default"]; h != nil {
			err := h(c.payload, cmd)
			if err != nil {
				c.msgr.SendError(c.payload, err)
			}
			return
		}

		c.msgr.Send(c.payload, "You must enter a command.\n")
		c.sendHelp()
		return
	}

	if cmd.Is("help") {
		c.sendHelp()
		return
	}

	for k := range c.handlers {
		if cmd.Is(k) {
			err := c.handlers[k](c.payload, cmd)
			if err != nil {
				c.msgr.SendError(c.payload, err)
			}
			return
		}
	}

	c.msgr.Send(c.payload, "Invalid command *"+cmd.Command+"*\n")
	c.sendHelp()
}

func (c *CmdHandler) sendHelp() {
	s := "*Usage:* `!" + c.name + " <command>`\n"
	if len(c.handlers) > 0 {
		cmds := ""
		for k := range c.handlers {
			if k == "_default" {
				continue
			}

			if cmds != "" {
				cmds += ", "
			}
			cmds += "`" + k + "`"
		}

		s += "*Commands:* " + cmds + "\n"
	}
	c.msgr.Send(c.payload, s)
}

//--------------

type Command struct {
	Command   string
	Arguments []string
	Params    map[string]string
}

func NewCommand(c string) Command {
	params := map[string]string{}
	args := []string{}
	parts := strings.Split(c, " ")

	cmd := parts[0]
	parts = append(parts[:0], parts[1:]...)

	for len(parts) > 0 {
		p := parts[0]
		parts = append(parts[:0], parts[1:]...)

		r := strings.Split(p, ":")
		if len(r) > 1 {
			params[r[0]] = r[1]
		} else {
			args = append(args, r[0])
		}
	}

	return Command{Command: cmd, Arguments: args, Params: params}
}

func (c *Command) Arg(idx int) string {
	if len(c.Arguments) > idx {
		return c.Arguments[idx]
	}

	return ""
}

func (c *Command) HasArgs() bool {
	return len(c.Arguments) > 0
}

func (c *Command) Param(s string) string {
	return c.Params[s]
}

func (c *Command) IsDefault() bool {
	return c.Command == ""
}

func (c *Command) Is(cmds ...string) bool {
	for _, cmd := range cmds {
		if c.Command == cmd {
			return true
		}
	}

	return false
}

func (c *Command) StrFrom(from int) string {
	return strings.Join(c.ArgsFrom(from), " ")
}

func (c *Command) ArgsFrom(from int) []string {
	args := []string{}
	for idx, a := range c.Arguments {
		if idx+1 > from {
			args = append(args, a)
		}
	}
	return args
}

func (c *Command) ParseArgs(args ...string) ([]string, error) {
	errMsg := ""
	res := []string{}
	for i, s := range args {
		if c.Arg(i) == "" {
			errMsg = "Missing *" + s + "*."
			break
		}
		res = append(res, c.Arg(i))
	}

	if errMsg != "" {
		errMsg += " Use `!" + c.Command + " "
		for _, s := range args {
			errMsg += "<" + s + "> "
		}
		errMsg += "`"
		return nil, errors.New(errMsg)
	}

	return res, nil
}
