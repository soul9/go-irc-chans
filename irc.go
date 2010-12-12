package ircchans

import (
	"os"
	"net"
	"log"
	"fmt"
	"bytes"
	"strings"
	"bufio"
	"time"
)

const (
	VERSION = "go-ircfs v0.0"
)

type Network struct {
	nick              string
	user              string
	network           string
	server            string
	realname          string
	password          string
	queueOut          chan string
	queueIn           chan string
	l                 *log.Logger
	conn              net.Conn
	done              chan bool
	QuitCh		chan bool
	buf               *bufio.ReadWriter
	ticker1, ticker15 <-chan int64
	listen            map[string]map[string]chan *IrcMessage //wildcard * is for any message
}

type IrcMessage struct {
	Prefix string
	Cmd    string
	Params []string
}

func (m *IrcMessage) String() string {
	if len(m.Params) > 15 {
		return ""
	}
	if m.Cmd == "" {
		return ""
	}
	msg := bytes.NewBufferString("")
	if m.Prefix != "" {
		msg.WriteString(fmt.Sprintf(":%s ", m.Prefix))
	}
	msg.WriteString(fmt.Sprintf("%s", m.Cmd))

	msg.WriteString(strings.Join(m.Params, " "))
	if msg.Len() > 510 {
		return ""
	}
	return msg.String()
}

func (n *Network) Connect() os.Error {
	if n.user == "" || n.nick == "" || n.realname == "" {
		return os.NewError("Empty nick and/or user and/or real name")
	}
	n.l.Printf("Connecting to irc network %s.\n", n.network)
	var err os.Error
	n.conn, err = net.Dial("tcp", "", n.network)
	if err != nil {
		return os.NewError(fmt.Sprintf("Couldn't connect to network %s.\n", n.network))
	}
	n.buf = bufio.NewReadWriter(bufio.NewReader(n.conn), bufio.NewWriter(n.conn))
	n.server = n.conn.RemoteAddr().String()
	n.l.Printf("Connected to network %s, server %s\n", n.network, n.server)
	n.ticker1 = time.Tick(1000 * 1000 * 1000 * 60 * 1)   //Tick every minute.
	n.ticker15 = time.Tick(1000 * 1000 * 1000 * 60 * 15) //Tick every 15 minutes.
	go n.overlook()
	go n.sender()
	go n.receiver()
	go n.pinger(nil, 0)
	go n.ponger(nil)
	go n.ctcp()
	if n.password != "" {
		n.Pass()
	}
	n.nick = n.Nick(n.nick)
	n.user = n.User(n.user)
	return nil
}

func (n *Network) overlook() {
	<- n.done  //wait for receiver and sender to quit
	<- n.done
	n.QuitCh <- true  //should reconnect here
	return
}

func (n *Network) sender() {
	if n.conn == nil {
		n.l.Println("socket closed, returning from sender()")
		n.done <- true
		return
	}
	if closed(n.queueOut) {
		n.l.Println("queueOut closed, returning from sender()")
		n.done <- true
		return
	}
	msg := <-n.queueOut
	_, err := n.buf.WriteString(fmt.Sprintf("%s\r\n", msg))
	if err != nil {
		n.l.Printf("Error writing to socket: %s, returning from sender()", err.String())
		n.done <- true
		return
	}
	err = n.buf.Flush()
	if err != nil {
		n.l.Printf("Error writing to socket: %s, returning from sender()", err.String())
		n.done <- true
		return
	}
	n.l.Printf(">>> %s\n", msg)
	n.sender()
}

func (n *Network) pinger(tick chan *IrcMessage, lastMessage int64) {
	if tick == nil {
		tick = make(chan *IrcMessage)
		n.RegListener("*", "ticker", tick)
	}
	select {
	case <-n.ticker1:
		n.l.Println("Ticked 1 minute")
		if time.Seconds()-lastMessage >= 60*4 {
			n.Ping()
		}
	case <-n.ticker15:
		//Ping every 15 minutes.
		n.l.Println("Ticked 15 minutes")
		n.Ping()
	case <-tick:
		n.l.Println("Don't tick for 4 minutes")
		lastMessage = time.Seconds()
	}
	n.pinger(tick, lastMessage)
}

func (n *Network) ponger(pingch chan *IrcMessage) {
	if pingch == nil {
		pingch = make(chan *IrcMessage)
		n.RegListener("PING", "ponger", pingch)
	}
	p := <-pingch
	if p == nil {
		n.l.Println("Something bad happened, ponger returning")
		n.DelListener("PING", "ponger")
		return
	}
	n.l.Printf("<<< %#v", p)
	n.Pong(p.Params[0])
	n.ponger(pingch)
}

//CTCP sucks, each client implements it a bit differently
func (n *Network) ctcp() {
	ch := make(chan *IrcMessage)
	n.RegListener("PRIVMSG", "ctcp", ch)
	for !closed(ch) {
		p := <-ch
		if i := strings.LastIndex(p.Params[1], "\x01"); i > -1{
			ctype := p.Params[1][2:i]
			dst := strings.Split(p.Prefix, "!", -1)[0]
			n.l.Println("<<< CTCP", p)
			switch {
			case  ctype == "VERSION":
				n.Notice(dst, fmt.Sprintf("\x01VERSION %s\x01", VERSION))
			case  ctype== "USERINFO":
				n.Notice(dst, fmt.Sprintf("\x01USERINFO %s\x01", n.user))
			case  ctype == "CLIENTINFO":
				n.Notice(dst, "\x01CLIENTINFO PING VERSION TIME USERINFO CLIENTINFO\x01")
			case  ctype[0:4] == "PING":
				params := strings.Split(p.Params[1], " ", -1)
				if len(params) < 2 {
					n.l.Println("Illegal ctcp ping received: No arguments", p)
					break
				}
				n.Notice(dst, fmt.Sprintf("\x01PING %s\x01", strings.Join(params[1:], " ")))
			case  ctype == "TIME":
				n.Notice(dst, fmt.Sprintf("\x01TIME %s\x01", time.LocalTime().String()))
			//TODO: ACTION, FINGER, SOURCE, PAGE?
			}
		}
	}
	n.l.Println("Something bad happened, ctcp returning")
	n.DelListener("PRIVMSG", "ctcp")
	return
}

func (n *Network) receiver() {
	if n.conn == nil {
		n.l.Println("Can't receive on socket: socket disconnected")
		n.done <- true
		return
	}
	l, err := n.buf.ReadString('\n')
	if err != nil {
		n.l.Println("Can't receive on socket: ", err.String())
		n.done <- true
		return
	}
	l = strings.TrimRight(l, "\r\n")
	msg := PackMsg(l)
	n.l.Printf("<<< %#v", msg)
	//dispatch
	go func() {
		for na, ch := range n.listen[msg.Cmd] {
			n.l.Printf("Delivering msg to %s", na)
			ch <- &msg
		}
		for na, ch := range n.listen["*"] {
			n.l.Printf("Delivering msg to %s", na)
			ch <- &msg
		}
		return
	}()
	n.receiver()
}

func PackMsg(msg string) IrcMessage {
	var ret IrcMessage
	if strings.HasPrefix(msg, ":") {
		if i := strings.Index(msg, " "); i > -1 {
			ret.Prefix = msg[1:i]
			msg = msg[i+1:]
		} else {
			log.Println("Malformed message: ", msg)
		}
	}
	if i := strings.Index(msg, " "); i > -1 {
		ret.Cmd = msg[0:i]
		msg = msg[i+1:]
	}
	ret.Params = strings.Split(msg, " ", -1)
	for i, m := range ret.Params {
		if strings.HasPrefix(m, ":") {
			ret.Params[i] = strings.Join(ret.Params[i:], " ")
			ret.Params = ret.Params[:i+1]
			break
		}
	}
	return ret
}

func (n *Network) RegListener(cmd, name string, ch chan *IrcMessage) os.Error {
	if _, ok := n.listen[cmd]; !ok {
		n.listen[cmd] = make(map[string]chan *IrcMessage)
	} else if _, ok := n.listen[cmd][name]; ok {
		return os.NewError(fmt.Sprintf("Can't register listener %s for cmd %s: already listening", name, cmd))
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

func NewNetwork(net, nick, usr, rn, pass, logfp string) *Network {
	n := new(Network)
	n.network = net
	n.password = pass
	n.nick = nick
	n.user = usr
	n.realname = rn
	n.listen = make(map[string]map[string]chan *IrcMessage)
	n.done = make(chan bool)
	n.QuitCh = make(chan bool)
	n.queueOut = make(chan string, 100)
	n.queueIn = make(chan string, 100)
	logflags := log.Ldate | log.Lmicroseconds | log.Llongfile
	logprefix := fmt.Sprintf("%s ", n.network)
	if logfp == "" {
		n.l = log.New(os.Stderr, logprefix, logflags)
	} else {
		f, err := os.Open(logfp, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
		if err != nil {
			n.l = log.New(os.Stderr, logprefix, logflags)
			n.l.Printf("Bad logfile: %s: %s\n", logfp, err.String())
		} else {
			n.l = log.New(f, logprefix, logflags)
		}
	}
	return n
}
