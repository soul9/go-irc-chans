package ircchans

import (
	"os"
	"fmt"
)

func (n *Network) RegListener(cmd, name string, ch chan *IrcMessage) os.Error {
	n.listen.lock.Lock()
	defer func(n *Network) { n.listen.lock.Unlock(); return }(n)
	if _, ok := n.listen.chans[cmd]; !ok {
		n.listen.chans[cmd] = make(map[string]chan *IrcMessage)
	} else if ch, ok := n.listen.chans[cmd][name]; ok {
		if ch != nil {
			return os.NewError(fmt.Sprintf("Can't register listener %s for cmd %s: already listening", name, cmd))
		}
	}
	n.listen.chans[cmd][name] = ch
	return nil
}

func (n *Network) DelListener(cmd, name string) os.Error {
	n.listen.lock.Lock()
	defer func(n *Network) { n.listen.lock.Unlock(); return }(n)
	if n.listen.chans[cmd] == nil || n.listen.chans[cmd][name] == nil {
		return os.NewError(fmt.Sprintf("No such listener: %s for cmd %s", name, cmd))
	}
	if !closed(n.listen.chans[cmd][name]) {
		close(n.listen.chans[cmd][name])
	}
	n.listen.chans[cmd][name] = nil, false
	if len(n.listen.chans[cmd]) == 0 {
		n.listen.chans[cmd] = nil, false
	}
	return nil
}
