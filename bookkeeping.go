package ircchans

import (
	"os"
	"fmt"
)

func (n *Network) RegListener(cmd, name string, ch chan *IrcMessage) os.Error {
	n.listen.lock.Lock()
	defer func(m dispatchMap) { m.lock.Unlock(); return }(n.listen)
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
	defer func(m dispatchMap) { m.lock.Unlock(); return }(n.listen)
	if n.listen.chans[cmd] == nil || n.listen.chans[cmd][name] == nil {
		return os.NewError(fmt.Sprintf("No such listener: %s for cmd %s", name, cmd))
	}
	if closed(n.listen.chans[cmd][name]) {
		n.listen.chans[cmd][name] = nil, false
		return os.NewError(fmt.Sprintf("Already closed; wiped: %s for cmd %s", name, cmd))
	}
	close(n.listen.chans[cmd][name])
	n.listen.chans[cmd][name] = nil, false
	return nil
}
