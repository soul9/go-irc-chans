package ircchans

import (
	"time"
)

func (n *Network) pinger() {
	tick := make(chan *IrcMessage)
	var lastMessage int64
	n.RegListener("*", "ticker", tick)
	for !closed(n.ticker1) && !closed(n.ticker15) && !closed(tick) {
		select {
		case <-n.ticker1:
			n.l.Println("Ticked 1 minute")
			if time.Seconds()-lastMessage >= 60*4 {
				n.Ping()
			}
		case <-n.ticker15:
			//Ping every 15 minutes.
			n.l.Println("Ticked 15 minutes")
			n.Ping()
		case <-tick:
			n.l.Println("Don't tick for 4 minutes")
			lastMessage = time.Seconds()
		}
	}
	n.l.Println("Something went terribly wrong, pinger exiting")
	n.DelListener("*", "ticker") //close channel and delete listener
	return
}

func (n *Network) ponger() {
	pingch := make(chan *IrcMessage)
	n.RegListener("PING", "ponger", pingch)
	for !closed(pingch) {
		p := <-pingch
		if p == nil {
			n.l.Println("Something bad happened, ponger returning")
			n.DelListener("PING", "ponger")
			return
		}
		n.Pong(p.Params[0])
	}
	n.l.Println("Something went terribly wrong, ponger exiting")
	return
}
