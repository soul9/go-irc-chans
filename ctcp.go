package ircchans

import (
	"strings"
	"fmt"
	"time"
)
//CTCP sucks, each client implements it a bit differently
func (n *Network) ctcp() {
	ch := make(chan *IrcMessage)
	n.Listen.RegListener("PRIVMSG", "ctcp", ch)
	for !closed(ch) {
		p := <-ch
		if i := strings.LastIndex(p.Params[1], "\x01"); i > -1 {
			ctype := p.Params[1][2:i]
			dst := strings.Split(p.Prefix, "!", -1)[0]
			switch {
			case ctype == "VERSION":
				n.Notice(dst, fmt.Sprintf("\x01VERSION %s\x01", VERSION))
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
	n.l.Println("Something bad happened, ctcp returning")
	n.Listen.DelListener("PRIVMSG", "ctcp")
	return
}
