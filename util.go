package ircchans

import (
	"time"
)

func (n *Network) pinger() {
	tick := make(chan *IrcMessage)
	var lastMessage int64
	n.Listen.RegListener("*", "ticker", tick)
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
	n.Listen.DelListener("*", "ticker") //close channel and delete listener
	return
}

func (n *Network) ponger() {
	pingch := make(chan *IrcMessage)
	n.Listen.RegListener("PING", "ponger", pingch)
	for !closed(pingch) {
		p := <-pingch
		if p == nil {
			n.l.Println("Something bad happened, ponger returning")
			break
		}
		n.Pong(p.Params[0])
	}
	n.Listen.DelListener("PING", "ponger")
	n.l.Println("Something went terribly wrong, ponger exiting")
	return
}

func (n *Network) logger() {
	inch := make(chan *IrcMessage, 10)
	outch := make(chan *IrcMessage, 10)
	n.Listen.RegListener("*", "logger", inch)
	n.OutListen.RegListener("*", "logger", outch)
	for {
		select {
		case m := <-inch:
			n.l.Printf("<<< %s", m.String())
		case m := <-outch:
			n.l.Printf(">>> %s", m.String())
		}
	}
	n.Listen.DelListener("*", "logger")
	n.OutListen.DelListener("*", "logger")
	return
}
