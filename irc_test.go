package ircchans

import (
	"testing"
	"time"
	"rand"
	"fmt"
	"strings"
)

var maxchans = 8 //BUG: (join doesn't return error, should handle when too many chans) join maximum number of chans on testing server

func runAsync(f func(*testing.T, *Network, []string), t *testing.T, n *Network, tchs []string, ch chan bool) {
	f(t, n, tchs)
	ch <- true
	return
}

func joinTests(t *testing.T, n *Network, tchs []string) {
	n.Ping()
	//join two channels (shouldn't be too many)
	if err := n.Join(tchs[len(tchs)-2:], []string{}); err != nil {
		t.Errorf("Join error: tried to join channels %#v, got error %s", tchs[len(tchs)-2:], err.String())
	}
	n.Part(tchs[1:2], "Gone phishing")
	n.Ping()
	//join too many chans
	if err := n.Join(tchs, []string{}); err == nil {
		t.Errorf("Join error: tried to join too many channels (%#v), didn't get any error", tchs)
	}
	n.Part(tchs, "Gone phishing")
	n.Ping()
	for i, _ := range tchs {
		if err := n.Join(tchs[i:i+1], []string{}); err != nil {
			t.Errorf("Join error: tried to join channel %s, got error %s", tchs[i:i+1], err.String())
		}
		n.Part(tchs[i:i+1], "Gone phishing")
		time.Sleep(second)
	}
	return
}

func privmsgStressTests(t *testing.T, n *Network, tchs []string) {
	if err := n.Join(tchs[len(tchs)-1:], []string{}); err != nil {
		t.Errorf("Join error: tried to join channel %s, got error %s", tchs[:1], err.String())
	}
	donech := make(chan bool)
	done := 0
	//stress-test
	numtests := 2000
	parallel := 30
	msgfmt := "Stress-testing %d"
	nick, _ := n.Nick("")
	msgparamfmt := fmt.Sprintf("%s %s", nick, msgfmt)
	testfunc := func(dch chan bool, n *Network, tchs []string, i int) {
		ch := make(chan *IrcMessage, parallel+10)
		if err := n.Listen.RegListener("PRIVMSG", fmt.Sprintf("testprivmsg%d", i), ch); err != nil {
			t.Errorf("Register error: tried to register cmd %s, name: %s got error %s", "PRIVMSG", fmt.Sprintf("testprivmsg%d", i), err.String())
		}
		go n.Privmsg([]string{nick}, fmt.Sprintf(msgfmt, i))
		timeout := time.NewTicker(minute / 2)
		for {
			select {
			case msg := <-ch:
				if strings.Join(msg.Params, " ") == fmt.Sprintf(msgparamfmt, i) {
					err := n.Listen.DelListener("PRIVMSG", fmt.Sprintf("testprivmsg%d", i))
					if err != nil {
						t.Errorf("Register error: tried to unregister cmd %s, name: %s got error %s", "PRIVMSG", fmt.Sprintf("testprivmsg%d", i), err.String())
					}
					dch <- true
					timeout.Stop()
					return
				}
			case <-timeout.C:
				t.Errorf("Stress-test error: didn't receive sent message: %d", i)
				err := n.Listen.DelListener("PRIVMSG", fmt.Sprintf("testprivmsg%d", i))
				if err != nil {
					t.Errorf("Register error: tried to unregister cmd %s, name: %s got error %s", "PRIVMSG", fmt.Sprintf("testprivmsg%d", i), err.String())
				}
				timeout.Stop()
				dch <- true
				return
			}
		}
	}
	for i := 0; i <= parallel; i++ {
		go testfunc(donech, n, tchs, i)
	}
	for {
		<-donech
		done++
		if numtests-done > parallel {
			go testfunc(donech, n, tchs, done+parallel)
		}
		if done%50 == 0 {
			n.Ping()
		}
		if done == numtests {
			return
		}
	}
}

func privmsgTests(t *testing.T, n *Network, tchs []string) {
	//test privmsg to too many recipients
	if err := n.Privmsg(tchs, "testing ☺"); err == nil {
		t.Errorf("Privmsg error: tried to send message to too many nicks (%#v), didn't get an error", tchs)
	}
	//try privmsg to unjoined channel
	if err := n.Privmsg(tchs[0:2], "testing ☺"); err == nil {
		t.Errorf("Privmsg error: tried to send message to unjoined channel (%#v), didn't get an error", tchs[0:2])
	}
	//test single privmsgs
	for _, ch := range tchs {
		err := n.Privmsg([]string{ch}, "Testing ☺")
		if err != nil {
			t.Errorf("Privmsg error: tried to send message to %s, got error %s", ch, err.String())
		}
	}
	return
}

func TestIrc(t *testing.T) {
	rand.Seed(time.Nanoseconds())
	testChans := func() []string {
		ret := make([]string, maxchans)
		retint := make([]int, maxchans)
		for i := 0; i < maxchans; i++ {
			for retint[i] == 0 {
				retint[i] = rand.Intn(8) //on test server channels go from 1 to 7)
			}
		}
		for i := 0; i < maxchans; i++ {
			ret[i] = fmt.Sprintf("#test%d", retint[i])
		}
		ret = append(ret, "#soul9")
		return ret
	}()
	//	network := "irc.didntdoit.net:16697"
	network := "irc.r0x0r.me:6667"
	nick := "ircchantest"
	user := "nottelling"
	realname := "I simply rock"
	password := "justpassingby"
	logfile := ""
	n := NewNetwork(network, nick, user, realname, password, logfile)
	if err := n.Connect(); err != nil {
		t.Errorf("Connect error: tried to connect to %s, got error %s", network, err.String())
		return
	}
	done := make(chan bool)
	jobs := 0
	n.Ping()
	go runAsync(joinTests, t, n, testChans, done)
	<-done
	n.Ping()
	go runAsync(privmsgTests, t, n, testChans, done)
	<-done
	n.Ping()
	go runAsync(privmsgStressTests, t, n, testChans, done)
	jobs++
	for jobs > 0 {
		<-done
		jobs--
	}

	if err := n.Reconnect("You're no fun anymore"); err != nil {
		t.Errorf("Reconnect error: tried to connect to %s, got error %s", network, err.String())
		return
	}

	n.Disconnect("You're no fun anymore") //TODO err?
}
//TODO: test ctcp(?), ping, ..
