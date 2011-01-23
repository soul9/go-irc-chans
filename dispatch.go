package ircchans

import (
	"os"
	"fmt"
	"sync"
)

type dispatchMap struct {
	lock  *sync.RWMutex
	chans map[string]map[string]chan *IrcMessage //wildcard * is for any message
}

func (m *dispatchMap) RegListener(cmd, name string, ch chan *IrcMessage) os.Error {
	m.lock.Lock()
	defer m.lock.Unlock()
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
	defer m.lock.Unlock()
	if m.chans[cmd] == nil || m.chans[cmd][name] == nil {
		return os.NewError(fmt.Sprintf("No such listener: %s for cmd %s", name, cmd))
	}
	if !closed(m.chans[cmd][name]) {
		if msg, ok := <-m.chans[cmd][name]; msg != nil || ok {
			close(m.chans[cmd][name])
		}
	}
	m.chans[cmd][name] = nil, false
	if len(m.chans[cmd]) == 0 {
		m.chans[cmd] = nil, false
	}
	return nil
}

func (m *dispatchMap) dispatch(msg IrcMessage) {
	m.lock.RLock()
	for _, ch := range m.chans[msg.Cmd] {
		_ = ch <- &msg
	}
	for _, ch := range m.chans["*"] {
		_ = ch <- &msg
	}
	m.lock.RUnlock()
	return
}

type shutdownDispatcher struct {
	lock    *sync.Mutex
	clients []chan bool
}

func (s *shutdownDispatcher) Reg(ch chan bool) os.Error {
	if len(ch) > 0 {
		return os.NewError("Need to pass synchronous channel for shutdown")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.clients = append(s.clients, ch)
	return nil
}

func (s *shutdownDispatcher) do() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	stale := 0
	for i, _ := range s.clients {
		ok := s.clients[i] <- true
		if !ok {
			stale++
		}
	}
	s.clients = make([]chan bool, 0)
	return stale
}
