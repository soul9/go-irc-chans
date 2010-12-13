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
	Disconnected      bool
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
	msg.WriteString(fmt.Sprintf("%s ", m.Cmd))

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
	n.Disconnected = false
	n.l.Printf("Connected to network %s, server %s\n", n.network, n.server)
	n.ticker1 = time.Tick(1000 * 1000 * 1000 * 60 * 1)   //Tick every minute.
	n.ticker15 = time.Tick(1000 * 1000 * 1000 * 60 * 15) //Tick every 15 minutes.
	go n.overlook()
	go n.sender()
	go n.receiver()
	go n.pinger()
	go n.ponger()
	go n.ctcp()
	if n.password != "" {
		n.Pass()
	}
	n.nick = n.Nick(n.nick)
	n.user = n.User(n.user)
	return nil
}

func (n *Network) Reconnect() os.Error {
	for _, ok := <- n.queueOut; ok; _, ok = <- n.queueOut {
		continue  //empty the queues
	}
	var err os.Error
	if n.conn != nil {
		n.conn.Close()
	}
	n.conn, err = net.Dial("tcp", "", n.network)
	if err != nil {
		return os.NewError(fmt.Sprintf("Couldn't connect to network %s: %s.\n", n.network, err.String()))
	}
	n.buf = bufio.NewReadWriter(bufio.NewReader(n.conn), bufio.NewWriter(n.conn))
	n.server = n.conn.RemoteAddr().String()
	n.Disconnected = false
	go n.sender()
	go n.receiver()
	if n.password != "" {
		n.Pass()
	}
	n.nick = n.Nick(n.nick)
	n.user = n.User(n.user)
	return nil
}

func (n *Network) sender() {
	timeout := time.NewTicker(1000 * 1000 * 1000 * 1)
	defer func(ch chan bool, t *time.Ticker) { t.Stop(); ch <- true }(n.done, timeout)
	for !n.Disconnected {
		if n.conn == nil {
			n.l.Println("socket closed, returning from sender()")
			n.Disconnected = true
			return
		}
		var msg string
		select {
		case msg = <-n.queueOut:
		case <-timeout.C: //timeout every second and check if we are disconnected
			if n.Disconnected {
				return
			} else {
				continue
			}
		}
		err, _ := PackMsg(msg)
		if err == nil {
			_, err = n.buf.WriteString(fmt.Sprintf("%s\r\n", msg))
			if err != nil {
				n.l.Printf("Error writing to socket: %s, returning from sender()", err.String())
				n.Disconnected = true
				return
			}
		} else {
			n.l.Println("Couldn't send malformed message: ", msg)
			continue
		}
		err = n.buf.Flush()
		if err != nil {
			n.l.Printf("Error writing to socket: %s, returning from sender()", err.String())
			n.Disconnected = true
			return
		}
		n.l.Printf(">>> %s\n", msg)
	}
	n.l.Println("Something went terribly wrong, sender exiting")
	return
}
func (n *Network) receiver() {
	defer func(ch chan bool) { ch <- true }(n.done)
	for !n.Disconnected {
		if n.conn == nil {
			n.l.Println("Can't receive on socket: socket disconnected")
			n.Disconnected = true
			return
		}
		l, err := n.buf.ReadString('\n')
		if err != nil {
			n.l.Println("Can't read: socket: ", err.String())
			n.Disconnected = true
			return
		}
		l = strings.TrimRight(l, "\r\n")
		err, msg := PackMsg(l)
		if err != nil {
			n.l.Printf("Couldn't unpack message: %s: %s", err.String(), l)
		}
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
		n.l.Printf("<<< %s", msg.String())
	}
	n.l.Println("Something went terribly wrong, receiver exiting")
	return
}

func PackMsg(msg string) (os.Error, IrcMessage) {
	var ret IrcMessage
	err := "Errors encountered during message packing: "
	if strings.HasPrefix(msg, ":") {
		if i := strings.Index(msg, " "); i > -1 {
			ret.Prefix = msg[1:i]
			msg = msg[i+1:]
		} else {
			err += "Malformed message, "
		}
	}
	if i := strings.Index(msg, " "); i > -1 {
		ret.Cmd = msg[0:i]
		msg = msg[i+1:]
	} else {
		err += "No command found"
	}
	ret.Params = strings.Split(msg, " ", -1)
	for i, m := range ret.Params {
		if strings.HasPrefix(m, ":") {
			ret.Params[i] = strings.Join(ret.Params[i:], " ")
			ret.Params = ret.Params[:i+1]
			break
		}
	}
	if err != "Errors encountered during message packing: " {
		return os.NewError(err), ret
	}
	return nil, ret
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
