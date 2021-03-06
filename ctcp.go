package ircchans

import (
	"strings"
	"fmt"
	"time"
)

//CTCP sucks, each client implements it a bit differently
func (n *Network) ctcp() {
	exch := make(chan bool, 0)
	err := n.Shutdown.Reg(exch)
	if err != nil {
		return
	}
	ch := make(chan *IrcMessage)
	n.Listen.RegListener("PRIVMSG", "ctcp", ch)
	defer n.Listen.DelListener("PRIVMSG", "ctcp")
	for !closed(ch) {
		var p *IrcMessage
		select {
		case p = <-ch:
		case exit := <-exch:
			if exit {
				return
			}
			continue
		}
		if i := strings.LastIndex(p.Params[1], "\x01"); i > -1 { //FIXME: DCC?
			p.Params[1] = strings.Trim(p.Params[1], "\x01")
			ctype := p.Params[1]
			dst := strings.Split(p.Prefix, "!", 2)[0]
			switch {
			case ctype == "VERSION":
				n.Notice(dst, fmt.Sprintf("\x01VERSION %s\x01", IRCVERSION))
			case ctype == "USERINFO":
				n.Notice(dst, fmt.Sprintf("\x01USERINFO %s\x01", n.user))
			case ctype == "CLIENTINFO":
				n.Notice(dst, "\x01CLIENTINFO PING VERSION TIME USERINFO CLIENTINFO FINGER SOURCE\x01")
			case ctype[0:4] == "PING":
				params := strings.Split(p.Params[1], " ", -1)
				if len(params) < 2 {
					n.l.Println("Illegal ctcp ping received: No arguments", p)
					break
				}
				n.Notice(dst, fmt.Sprintf("\x01PING %s\x01", strings.Join(params[1:], " ")))
			case ctype == "TIME":
				n.Notice(dst, fmt.Sprintf("\x01TIME %s\x01", time.LocalTime().String()))
				//TODO: ACTION, PAGE?
			case ctype == "FINGER":
				n.Notice(dst, fmt.Sprintln("\x01FINGER like i'm gonna tell you\x01"))
			case ctype == "SOURCE":
				n.Notice(dst, fmt.Sprintln("\x01SOURCE https://github.com/soul9/go-irc-chans\x01"))

			}
		}
	}
	return
}

func (n *Network) CtcpVersion(target string) {
}

func (n *Network) CtcpUserInfo(target string) {
}

func (n *Network) CtcpClientInfo(target string) {
}

func (n *Network) CtcpPing(target string) {
}

func (n *Network) CtcpTime(target string) {
}

func (n *Network) CtcpFinger(target string) {
}

func (n *Network) CtcpSource(target string) {
}

func (n *Network) CtcpAction(target string) {
}
