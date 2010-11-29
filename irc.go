package main

import (
	"os"
	"net"
	"log"
	"fmt"
	"bytes"
	"strings"
	"flag"
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
	buf               *bufio.ReadWriter
	ticker1, ticker15 <-chan int64
	listen            map[string]map[string]chan IrcMessage //wildcard * is for any message
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
	n.listen = make(map[string]map[string]chan IrcMessage)
	n.l.Printf("Connecting to irc network %s.\n", n.network)
	var err os.Error
	n.conn, err = net.Dial("tcp", "", n.network)
	if err != nil {
		return os.NewError(fmt.Sprintf("Couldn't connect to network %s.\n", n.network))
	}
	n.buf = bufio.NewReadWriter(bufio.NewReader(n.conn), bufio.NewWriter(n.conn))
	n.server = n.conn.RemoteAddr().String()
	n.l.Printf("Connected to network %s, server %s\n", n.network, n.server)
	n.done = make(chan bool)
	n.queueOut = make(chan string, 100)
	n.queueIn = make(chan string, 100)
	go n.sender()
	go n.pinger()
	go n.receiver()
	go n.ponger()
	go n.ctcp()
	if n.password != "" {
		n.Pass()
	}
	n.nick = n.Nick(n.nick)
	n.user = n.User(n.user)
	return nil
}

func (n *Network) sender() {
	for {
		if n.conn == nil {
			n.l.Println("socket closed, returning from sender()")
			break
		}
		if closed(n.queueOut) {
			n.l.Println("queueOut closed, returning from sender()")
			break
		}
		msg := <-n.queueOut
		_, err := n.buf.WriteString(fmt.Sprintf("%s\r\n", msg))
		if err != nil {
			n.l.Printf("Error writing to socket: %s, returning from sender()", err.String())
			break
		}
		err = n.buf.Flush()
		if err != nil {
			n.l.Printf("Error writing to socket: %s, returning from sender()", err.String())
			break
		}
		n.l.Printf("Message sent: %s\n", msg)
	}
	n.done <- true
	return
}

func (n *Network) pinger() {
	n.ticker1 = time.Tick(1000 * 1000 * 1000 * 60 * 1)   //Tick every minute.
	n.ticker15 = time.Tick(1000 * 1000 * 1000 * 60 * 15) //Tick every 15 minutes.
	tick := make(chan IrcMessage)
	n.RegListener("*", "ticker", tick)
	var lastMessage int64
	for {
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
	}
	return
}

func (n *Network) ponger() {
	pingch := make(chan IrcMessage)
	n.RegListener("PING", "ponger", pingch)
	for !closed(pingch) {
		p := <-pingch
		n.Pong(p.Params[0])
	}
	n.l.Println("Something bad happened, ponger returning")
	n.DelListener("PING", "ponger")
	return
}

func (n *Network) ctcp() {
	ch := make(chan IrcMessage)
	n.RegListener("PRIVMSG", "ctcp", ch)
	for !closed(ch) {
		p := <-ch
		if i := strings.LastIndex(p.Params[1], "\x01"); i > -1{
			ctype := p.Params[1][2:i]
			n.l.Println("recvd CTCP type", ctype)
			switch {
			case  ctype == "VERSION":
				n.Notice(p.Prefix, fmt.Sprintf("\x01VERSION %s\x01", VERSION))
			case  ctype== "USERINFO":
				n.Notice(p.Prefix, fmt.Sprintf("\x01USERINFO %s\x01", n.user))
			case  ctype == "CLIENTINFO":
				n.Notice(p.Prefix, "\x01CLIENTINFO PING VERSION TIME USERINFO CLIENTINFO\x01")
			case  ctype[0:4] == "PING":
				n.Notice(p.Prefix, fmt.Sprintf("\x01PING %s\x01", strings.Split(p.Params[1], " ", -1)[1][1:]))
			case  ctype == "TIME":
				n.Notice(p.Prefix, fmt.Sprintf("\x01TIME %s\x01", time.LocalTime().String()))
			}
		}
	}
	n.l.Println("Something bad happened, ponger returning")
	n.DelListener("PRIVMSG", "ctcp")
	return
}

func (n *Network) receiver() {
	for {
		if n.conn == nil {
			n.l.Println("Can't receive on socket: socket disconnected")
			break
		}
		l, err := n.buf.ReadString('\n')
		if err != nil {
			n.l.Println("Can't receive on socket: ", err.String())
			break
		}
		l = strings.TrimRight(l, "\r\n")
		msg := PackMsg(l)
		//dispatch
		go func() {
			for na, ch := range n.listen[msg.Cmd] {
				n.l.Printf("Delivering msg to %s", na)
				ch <- msg
			}
			for na, ch := range n.listen["*"] {
				n.l.Printf("Delivering msg to %s", na)
				ch <- msg
			}
			return
		}()
		n.l.Printf("Message received: %s msg: %#v", l, msg)
	}
	n.done <- true
	return
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

func (n *Network) RegListener(cmd, name string, ch chan IrcMessage) os.Error {
	if _, ok := n.listen[cmd]; !ok {
		n.listen[cmd] = make(map[string]chan IrcMessage)
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

func main() {
	netf := flag.String("net", "viotest.local:6667", "Network name in the form of network.dom:port")
	passf := flag.String("p", "", "Network Password")
	nickf := flag.String("n", "go-ircfs", "Nickname on network")
	userf := flag.String("u", "", "Irc user (defaults to nick)")
	rnf := flag.String("r", "go-ircfs", "Real Name (defaults to nick)")
	logfile := flag.String("l", "", "File used for logging (default: stderr)")
	usage := flag.Bool("h", false, "Display usage and help message")
	flag.Parse()
	if *usage {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *userf == "" {
		userf = nickf
	}
	if *rnf == "" {
		rnf = nickf
	}

	n := NewNetwork(*netf, *nickf, *userf, *rnf, *passf, *logfile)
	err := n.Connect()
	if err != nil {
		n.l.Println(err.String())
		os.Exit(1)
	}
	n.Join([]string{"#soul9", "#ubuntu", "#t"}, []string{})
	go func(){
		chin := make(chan IrcMessage, 100)
		n.RegListener("PRIVMSG", "testreply", chin)
		for !closed(chin) {
			msg := <- chin
			if msg.Params[0] == n.nick {
				n.Privmsg([]string{msg.Prefix}, strings.Join(msg.Params[1:], " "))
			} else {
				n.Privmsg(msg.Params[:1], strings.Join(msg.Params[1:], " "))
			}
		}
	}()
	<-n.done
	os.Exit(0)
}
