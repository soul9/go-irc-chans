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


func PackMsg(msg string) (os.Error, IrcMessage) { //TODO: this needs work?
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
		return os.NewError(err), ret
	}
	return nil, ret
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
	params := bytes.NewBufferString("")
	space := false
	for i, p := range m.Params {
		if idx := strings.Index(p, " "); idx < 0 {
			msg.WriteString(fmt.Sprintf(" %s", p))
		} else {
			space = true
			msg.WriteString(fmt.Sprintf(" :%s", strings.Join(m.Params[i:], " ")))
			break
		}
	}
	if space {
		msg.WriteString(params.String())
	} else {
		msg.WriteString(fmt.Sprintf(":%s", params.String()))
	}
	if msg.Len() > 510 {
		return ""
	}
	return msg.String()
}

func (m *IrcMessage) Origin() map[string]string {
	ret := make(map[string]string)
	if m.Prefix != "" {
		tmp := strings.Split(m.Prefix, "!", 2)
		ret["nick"] = tmp[0]
		tmp = strings.Split(tmp[1], "@", 2)
		ret["user"] = tmp[0]
		ret["host"] = tmp[1]
	} else {
		if m.Cmd == "PRIVMSG" {
			ret["chan"] = m.Params[0]
		}
	}
	return ret
}

func (m *IrcMessage) Destination() string {
	if m.Cmd == "PRIVMSG" {
		return m.Params[0]
	}
	return ""
}
