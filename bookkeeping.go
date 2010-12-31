package ircchans

import (
	"os"
	"fmt"
)

func (m *dispatchMap) RegListener(cmd, name string, ch chan *IrcMessage) os.Error {
	m.lock.Lock()
	defer func(m *dispatchMap) { m.lock.Unlock(); return }(m)
	if _, ok := m.chans[cmd]; !ok {
		m.chans[cmd] = make(map[string]chan *IrcMessage)
	} else if ch, ok := m.chans[cmd][name]; ok {
		if ch != nil {
			return os.NewError(fmt.Sprintf("Can't register listener %s for cmd %s: already listening", name, cmd))
		}
	}
	m.chans[cmd][name] = ch
	return nil
}

func (m *dispatchMap) DelListener(cmd, name string) os.Error {
	m.lock.Lock()
	defer func(m *dispatchMap) { m.lock.Unlock(); return }(m)
	if m.chans[cmd] == nil || m.chans[cmd][name] == nil {
		return os.NewError(fmt.Sprintf("No such listener: %s for cmd %s", name, cmd))
	}
	if !closed(m.chans[cmd][name]) {
		close(m.chans[cmd][name])
	}
	m.chans[cmd][name] = nil, false
	if len(m.chans[cmd]) == 0 {
		m.chans[cmd] = nil, false
	}
	return nil
}
