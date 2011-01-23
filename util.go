package ircchans

import (
	"time"
)

func (n *Network) pinger() {
	exch := make(chan bool, 0)
	err := n.Shutdown.Reg(exch)
	if err != nil {
		return
	}
	ticker1 := time.NewTicker(minute)
	defer ticker1.Stop()
	ticker15 := time.NewTicker(minute * 15)
	defer ticker15.Stop()
	tick := make(chan *IrcMessage)
	var lastMessage int64
	n.Listen.RegListener("*", "ticker", tick)
	defer n.Listen.DelListener("*", "ticker") //close channel and delete listener
	for {
		select {
		case <-ticker1.C:
			if time.Seconds()-lastMessage >= 60*4 { //ping about every five minutes if there is no activity at all
				n.Ping()
				n.l.Printf("Network lag is: %d nanoseconds", n.lag)
			}
		case <-ticker15.C:
			//Ping every 15 minutes.
			n.Ping()
			n.l.Printf("Network lag is: %d nanoseconds", n.lag)
		case <-tick:
			lastMessage = time.Seconds()
		case exit := <-exch:
			if exit {
				return
			}
			continue
		}
	}
	return
}

func (n *Network) ponger() {
	exch := make(chan bool, 0)
	err := n.Shutdown.Reg(exch)
	if err != nil {
		return
	}
	pingch := make(chan *IrcMessage)
	n.Listen.RegListener("PING", "ponger", pingch)
	defer n.Listen.DelListener("PING", "ponger")
	for !closed(pingch) {
		select {
		case p := <-pingch:
			if p == nil {
				n.l.Println("Something bad happened, ponger returning")
				n.Disconnect("Software error")
				return
			}
			n.Pong(p.Params[0])
		case exit := <-exch:
			if exit {
				return
			}
			continue
		}
	}
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
			n.l.Printf("<<< %#v", m)
		case m := <-outch:
			n.l.Printf(">>> %#v", m)
		}
	}
	n.Listen.DelListener("*", "logger")
	n.OutListen.DelListener("*", "logger")
	return
}
