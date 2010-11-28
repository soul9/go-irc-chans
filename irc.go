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
	nick string
	user string
	network string
	server string
	realname string
	password string
	queueOut chan string
	queueIn chan string
	l *log.Logger
	conn net.Conn
	done chan bool
	buf *bufio.ReadWriter
	ticker4, ticker15 <- chan int64
	listen map[string]map[string]chan IrcMessage //wildcard * is for any message
}

type IrcMessage struct {
	Prefix string
	Cmd string
	Params []string
}

func (m *IrcMessage) String() (string, os.Error) {
	if len(m.Params) > 15 {
		return "", os.NewError("Too many parameters. Maximum (according to rfc) is 15.")
	}
	if m.Cmd == "" {
		return "", os.NewError("Message command is empty.")
	}
	msg := bytes.NewBufferString("")
	if m.Prefix != "" {
		msg.WriteString(fmt.Sprintf(":%s ", m.Prefix))
	}
	msg.WriteString(fmt.Sprintf("%s", m.Cmd))

	msg.WriteString(strings.Join(m.Params, " "))
	if msg.Len() > 510 {
		return "", os.NewError("Message too long. Maximum length is 510 (according to rfc).")
	}
	return msg.String(), nil
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
		if closed(n.queueOut){
			n.l.Println("queueOut closed, returning from sender()")
			break
		} 
		msg := <- n.queueOut
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
	n.ticker4 = time.Tick(1000 * 1000 * 1000 * 60 * 4)   //Tick every minute.
	n.ticker15 = time.Tick(1000 * 1000 * 1000 * 60 * 15) //Tick every 15 minutes.
	tick := make(chan IrcMessage)
	n.RegListener("*", "ticker", tick)
	var lastMessage int64
	for {
		select {
		case <-n.ticker4:
			n.l.Println("Ticked 4 minutes")
			if time.Seconds()-lastMessage >= 60*4 {
				n.Ping()
			}
		case <-n.ticker15:
			//Ping every 15 minutes.
			n.l.Println("Ticked 15 minutes")
			n.Ping()
		case <- tick:
			n.l.Println("Don't tick now, tick in 4 minutes..")
			lastMessage = time.Seconds()
		}
	}
	return
}

func (n *Network) ponger() {
	pingch := make(chan IrcMessage)
	n.RegListener("PING", "PONGER", pingch)
	for !closed(pingch) {
		p := <- pingch
		n.Pong(p.Params[0])
	}
	n.l.Println("Something bad happened, ponger returning")
	n.DelListener("PING", "PONGER")
	return
}

func (n *Network) receiver() {
	n.l.Println("Receiver listening", n.buf)
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
		go func(){
			for _, ch := range n.listen[msg.Cmd] {
				ch <- msg
			}
			for _, ch := range n.listen["*"] {
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
		if i := strings.Index(msg, " "); i > -1{
			ret.Prefix = msg[1:i]
			msg = msg[i+1:]
		} else {
			log.Println("Malformed message: ", msg)
		}
	}
	if i := strings.Index(msg, " "); i > -1{
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
	if n.listen[cmd] == nil {
		n.listen[cmd] = make(map[string] chan IrcMessage)
	} else if n.listen[cmd][name] != nil || !closed(n.listen[cmd][name]) {
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
	logflags := log.Ldate|log.Lmicroseconds|log.Llongfile
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
	n.Join([]string{"#soul9", "#ubuntu"}, []string{})
	<- n.done
	os.Exit(0)
}
