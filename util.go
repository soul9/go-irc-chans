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
			if time.Seconds()-lastMessage >= 60*4 { //ping about every five minutes if there is no activity at all
				n.Ping()
				n.l.Printf("Network lag is: %d nanoseconds", n.lag)
			}
		case <-n.ticker15:
			//Ping every 15 minutes.
			n.Ping()
			n.l.Printf("Network lag is: %d nanoseconds", n.lag)
		case <-tick:
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
			break
		}
		n.Pong(p.Params[0])
	}
	n.DelListener("PING", "ponger")
	n.l.Println("Something went terribly wrong, ponger exiting")
	return
}
