package ircchans

import (
	"testing"
	"time"
	"rand"
	"fmt"
	"strings"
	"os"
)
//test server
import "bitbucket.org/kylelemons/jaid/src/pkg/irc"


const (
	maxchans   = 10
	clients    = 100
	sslclients = 20
	logfile    = "test/test.log"
)

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
	//join too many chans
	if err := n.Join(tchs, []string{}); err == nil {
		t.Errorf("Join error: tried to join too many channels (%#v), didn't get any error", tchs)
	}
	n.Part(tchs, "Gone phishing")
	for i, _ := range tchs {
		if err := n.Join(tchs[i:i+1], []string{}); err != nil {
			t.Errorf("Join error: tried to join channel %s, got error %s", tchs[i:i+1], err.String())
		}
		time.Sleep(second / 2)
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
	numtests := 400
	parallel := 2
	msgfmt := "Stress-testing %d"
	nick := n.GetNick()
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
		err := n.Privmsg([]string{ch}, "Testing☺")
		if err != nil {
			t.Errorf("Privmsg error: tried to send message to %s, got error %s", ch, err.String())
		}
	}
	return
}

func doIrcStuff(n *Network, t *testing.T, tchs []string, done chan bool) {
	err := n.Connect()
	net, _ := n.NetName("", "")
	if err != nil {
		t.Fatalf("Couldn't connect to network %s", net)
		os.Exit(1)
	}
	jobs := 0
	d := make(chan bool)
	go runAsync(joinTests, t, n, tchs, d)
	jobs++
	n.Ping()
	go runAsync(privmsgTests, t, n, tchs, d)
	jobs++
	go runAsync(privmsgStressTests, t, n, tchs, d)
	jobs++
	for jobs > 0 {
		<-d
		jobs--
	}

	if err := n.Reconnect("Testing reconnect"); err != nil {
		t.Fatalf("Reconnect error: tried to connect to %s, got error %s", net, err.String())
	}
	n.Disconnect("You're no fun anymore.")
	if err := n.Reconnect("Testing reconnect"); err != nil {
		t.Fatalf("Reconnect error: tried to connect to %s, got error %s", net, err.String())
	}

	go runAsync(joinTests, t, n, tchs, d)
	jobs++
	n.Ping()
	go runAsync(privmsgTests, t, n, tchs, d)
	jobs++
	go runAsync(privmsgStressTests, t, n, tchs, d)
	jobs++
	for jobs > 0 {
		<-d
		jobs--
	}
	n.Disconnect("You're no fun anymore.")
	done <- true
}

func TestIrc(t *testing.T) {
	go func() {
		srv := irc.NewServer("test/jaid/ircd.conf", "jaid-v0.0.1", "Today")
		srv.Run()
		t.Fatal("Test server shut down")
		os.Exit(1)
	}()
	time.Sleep(minute / 4) //wait for jaid to start up
	rand.Seed(time.Nanoseconds())

	testChans := func() []string {
		ret := make([]string, maxchans)
		retint := make([]int, maxchans)
		for i := 0; i < maxchans; i++ {
			for retint[i] == 0 {
				retint[i] = rand.Intn(400) //on test server channels go from 1 to 7)
			}
		}
		for i := 0; i < maxchans; i++ {
			ret[i] = fmt.Sprintf("#test%d", retint[i])
		}
		return ret
	}

	network := "localhost:16667"
	sslnetwork := "localhost:16697"
	nick := "test"
	user := "nottelling"
	realname := "I simply rock"
	password := "" //TODO: jaid+network pass
	cls := make([]*Network, clients)
	sslcls := make([]*Network, sslclients)
	jobs := 0
	done := make(chan bool)
	for i := 0; i < clients; i++ {
		cls[i] = NewNetwork(network, fmt.Sprintf("%s%d", nick, i), user, realname, password, logfile)
		go func(i int) {
			err := cls[i].Connect()
			if err != nil {
				t.Fatalf("Error connecting: %s: %s", cls[i].GetNick(), err.String())
			}
			done <- true
		}(i)
		jobs++
	}
	for i := 0; i < sslclients; i++ {
		sslcls[i] = NewNetwork(sslnetwork, fmt.Sprintf("%s%d", nick, i), user, realname, password, logfile)
		sslcls[i].Connect()
		go func(i int) {
			err := sslcls[i].Connect()
			if err != nil {
				t.Fatalf("Error connecting: %s: %s", sslcls[i].GetNick(), err.String())
			}
			done <- true
		}(i)
		jobs++
	}
	for jobs > 0 {
		<-done
		jobs--
	}

	for i := 0; i < clients; i++ {
		go doIrcStuff(cls[i], t, testChans(), done)
		jobs++
	}
	for i := 0; i < sslclients; i++ {
		go doIrcStuff(sslcls[i], t, testChans(), done)
		jobs++
	}
	for jobs > 0 {
		<-done
		jobs--
	}
}
//TODO: test ctcp(?), ping, ..
