package ircchans

import (
	"testing"
	"time"
	"rand"
	"fmt"
)

var maxchans = 8 //BUG: (join doesn't return error, should handle when too many chans) join maximum number of chans on testing server

func joinTests(t *testing.T, n *Network, tchs []string) {
	//join a single channel
	if err := n.Join(tchs[:1], []string{}); err != nil {
		t.Errorf("Join error: tried to join channel %s, got error %s", tchs[:1], err.String())
	}
	n.Part(tchs[:1], "Gone phishing")
	//join a single channel
	if err := n.Join(tchs[1:2], []string{}); err != nil {
		t.Errorf("Join error: tried to join channel %s, got error %s", tchs[1:2], err.String())
	}
	n.Part(tchs[1:2], "Gone phishing")
	//join two channels (shouldn't be too many)
	if err := n.Join(tchs[maxchans-2:], []string{}); err != nil {
		t.Errorf("Join error: tried to join channels %#v, got error %s", tchs[maxchans-2:], err.String())
	}
	n.Part(tchs[1:2], "Gone phishing")
	//join too many chans
	if err := n.Join(tchs, []string{}); err == nil {
		t.Errorf("Join error: tried to join too many channels (%#v), didn't get any error", tchs)
	}
	n.Part(tchs, "Gone phishing")
	for i, _ := range tchs {
		if err := n.Join(tchs[i:i+1], []string{}); err != nil {
			t.Errorf("Join error: tried to join channel %s, got error %s", tchs[i:i+1], err.String())
		}
		time.Sleep(second)
		n.Part(tchs[i:i+1], "Gone phishing")
		time.Sleep(second)
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
	network := "irc.r0x0r.me:6667"
	nick := "ircchantest"
	user := "nottelling"
	realname := "I simply rock"
	password := "justpassingby"
	logfile := "/dev/null"
	n := NewNetwork(network, nick, user, realname, password, logfile)
	if err := n.Connect(); err != nil {
		t.Errorf("Connect error: tried to connect to %s, got error %s", network, err.String())
		return
	}
	joinTests(t, n, testChans)
	//test privmsg to too many recipients
	if err := n.Privmsg(testChans, "testing ☺"); err == nil {
		t.Errorf("Privmsg error: tried to send message to too many nicks (%#v), didn't get an error", testChans)
	}
	//try privmsg to unjoined channel
	if err := n.Privmsg(testChans[0:2], "testing ☺"); err == nil {
		t.Errorf("Privmsg error: tried to send message to unjoined channel (%#v), didn't get an error", testChans[0:2])
	}
	//test single privmsgs
	for _, ch := range testChans {
		err := n.Privmsg([]string{ch}, "Testing ☺")
		if err != nil {
			t.Errorf("Privmsg error: tried to send message to %s, got error %s", ch, err.String())
		}
	}
	n.Disconnect("You're no fun anymore") //todo err?
}
