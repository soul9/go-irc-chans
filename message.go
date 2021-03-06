package ircchans

import (
	"os"
	"strings"
	"bytes"
	"fmt"
)

type IrcMessage struct {
	Prefix string
	Cmd    string
	Params []string
}


func PackMsg(msg string) (IrcMessage, os.Error) { //TODO: this needs work?
	var ret IrcMessage
	err := "Errors encountered during message packing: "
	if strings.HasPrefix(msg, ":") {
		if i := strings.Index(msg, " "); i > -1 {
			ret.Prefix = msg[1:i]
			msg = msg[i+1:]
		} else {
			err += "Malformed message, "
		}
	}
	if i := strings.Index(msg, " "); i > -1 {
		ret.Cmd = msg[0:i]
		msg = msg[i+1:]
	} else {
		err += "No command found, "
	}
	ret.Params = strings.Split(msg, " ", -1)
	for i, m := range ret.Params {
		if strings.HasPrefix(m, ":") {
			ret.Params[i] = ret.Params[i][1:]
			ret.Params[i] = strings.Join(ret.Params[i:], " ")
			ret.Params = ret.Params[:i+1]
			break
		}
	}
	if err != "Errors encountered during message packing: " {
		return ret, os.NewError(err)
	}
	return ret, nil
}

func (m *IrcMessage) String() string {
	if len(m.Params) > 15 {
		return ""
	}
	if m.Cmd == "" {
		return ""
	}
	msg := bytes.NewBufferString("")
	if m.Prefix != "" {
		msg.WriteString(fmt.Sprintf(":%s ", m.Prefix))
	}
	msg.WriteString(fmt.Sprintf("%s", m.Cmd))
	for i, p := range m.Params {
		if idx := strings.Index(p, " "); idx < 0 {
			msg.WriteString(fmt.Sprintf(" %s", p))
		} else {
			msg.WriteString(fmt.Sprintf(" :%s", strings.Join(m.Params[i:], " ")))
			break
		}
	}
	if msg.Len() > 510 {
		return ""
	}
	return msg.String()
}

func (m *IrcMessage) Origin() string {
	if m.Prefix != "" {
		return m.Prefix
	} else {
		if m.Cmd == "PRIVMSG" {
			return m.Params[0]
		}
	}
	return ""
}

func (m *IrcMessage) Destination() string {
	if m.Cmd == "PRIVMSG" {
		return m.Params[0]
	}
	return ""
}

func (m *IrcMessage) Payload() string {
	if m.Cmd == "PRIVMSG" {
		return strings.Join(m.Params[1:], " ")
	}
	return ""
}
