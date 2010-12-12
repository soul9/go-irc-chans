package ircchans

import (
	"os"
	"fmt"
	"time"
)

func (n *Network) RegListener(cmd, name string, ch chan *IrcMessage) os.Error {
	if _, ok := n.listen[cmd]; !ok {
		n.listen[cmd] = make(map[string]chan *IrcMessage)
	} else if ch, ok := n.listen[cmd][name]; ok {
		if ch != nil {
			return os.NewError(fmt.Sprintf("Can't register listener %s for cmd %s: already listening", name, cmd))
		}
	}
	n.listen[cmd][name] = ch
	return nil
}

func (n *Network) DelListener(cmd, name string) os.Error {
	if n.listen[cmd] == nil || n.listen[cmd][name] == nil {
		return os.NewError(fmt.Sprintf("No such listener: %s for cmd %s", name, cmd))
	}
	if closed(n.listen[cmd][name]) {
		n.listen[cmd][name] = nil
		return os.NewError(fmt.Sprintf("Already closed; wiped: %s for cmd %s", name, cmd))
	}
	close(n.listen[cmd][name])
	n.listen[cmd][name] = nil
	return nil
}

func (n *Network) overlook() {
	for {
		<- n.done  //wait for receiver and sender to quit
		<- n.done
		n.Disconnected = true
		n.l.Println("Something went wrong, reconnecting")
		err := n.Reconnect()
		if err != nil {
			for i := 0; i<=30; i++ {
				n.l.Printf("Error during reconnect: %s", err.String())
				<- time.Tick(1000 * 1000 * 1000 * 10)  //try reconnecting every 10 second
				err = n.Reconnect()
				if err == nil {
					break
				}
			}
		}
	}
	return
}