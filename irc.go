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
	"sync"
)

const (
	VERSION = "go-irc-chans v0.1"
	minute  = 1000 * 1000 * 1000 * 60
)

type Network struct {
	nick              string
	user              string
	network           string
	server            string
	realname          string
	password          string
	lag               int64
	queueOut          chan string
	queueIn           chan string
	l                 *log.Logger
	conn              net.Conn
	Disconnected      bool
	buf               *bufio.ReadWriter
	ticker1, ticker15 <-chan int64
	listen            dispatchMap
}

type IrcMessage struct {
	Prefix string
	Cmd    string
	Params []string
}

type dispatchMap struct {
	lock  *sync.RWMutex
	chans map[string]map[string]chan *IrcMessage //wildcard * is for any message
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
	var err os.Error
	for _, ok := <-n.queueOut; ok; _, ok = <-n.queueOut { //empty the write channel so we don't send out-of-context messages
		continue
	}
	if n.user == "" || n.nick == "" || n.realname == "" {
		return os.NewError("Empty nick and/or user and/or real name")
	}
	n.conn, err = net.Dial("tcp", "", n.network)
	if err != nil {
		return os.NewError(fmt.Sprintf("Couldn't connect to network %s: %s.\n", n.network, err.String()))
	}
	n.buf = bufio.NewReadWriter(bufio.NewReader(n.conn), bufio.NewWriter(n.conn))
	n.server = n.conn.RemoteAddr().String()
	n.Disconnected = false
	n.l.Printf("Connected to network %s, server %s\n", n.network, n.server)
	go n.receiver()
	if n.password != "" {
		n.Pass()
	}
	n.nick = n.Nick(n.nick)
	n.user = n.User(n.user)
	n.Ping()
	n.l.Printf("Network lag is: %d nanoseconds", n.lag)
	return nil
}

func (n *Network) Reconnect() os.Error {
	n.l.Printf("Connecting to irc network %s.\n", n.network)
	n.Disconnect("Reconnecting..")
	return n.Connect()
}

func (n *Network) Disconnect(reason string) {
	if n.conn != nil {
		n.Quit(reason)
		time.Sleep(minute / 60) //sleep 1 second to send QUIT message
		n.conn.Close()
	}
	n.Disconnected = true
	return
}

func (n *Network) sender() {
	for {
		msg := <-n.queueOut
		err, _ := PackMsg(msg)
		if err == nil {
			_, err = n.buf.WriteString(fmt.Sprintf("%s\r\n", msg))
			if err != nil {
				n.l.Printf("Error writing to socket (%s): %s", err.String(), msg)
				continue
			}
		} else {
			n.l.Println("Couldn't send malformed message: ", msg)
			continue
		}
		err = n.buf.Flush()
		if err != nil {
			n.l.Printf("Error flushing socket (%s): %s", err.String(), msg)
			continue
		}
		n.l.Printf(">>> %s\n", msg)
	}
	return
}

func (n *Network) receiver() {
	for {
		l, err := n.buf.ReadString('\n')
		if err != nil {
			n.l.Println("Can't read: socket: ", err.String())
			n.Disconnect("Connection error")
			return
		}
		l = strings.TrimRight(l, "\r\n")
		err, msg := PackMsg(l)
		if err != nil {
			n.l.Printf("Couldn't unpack message: %s: %s", err.String(), l)
		}
		//dispatch
		go func() {
			n.listen.lock.RLock()
			for _, ch := range n.listen.chans[msg.Cmd] {
				_ = ch <- &msg
			}
			for _, ch := range n.listen.chans["*"] {
				_ = ch <- &msg
			}
			n.listen.lock.RUnlock()
			return
		}()
		n.l.Printf("<<< %s", msg.String())
	}
	n.l.Println("Something went terribly wrong, receiver exiting")
	return
}

func PackMsg(msg string) (os.Error, IrcMessage) { //TODO: this needs work?
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
	n.listen = dispatchMap{new(sync.RWMutex), make(map[string]map[string]chan *IrcMessage)}
	n.queueOut = make(chan string, 100)
	n.queueIn = make(chan string, 100)
	n.ticker1 = time.Tick(minute)       //Tick every minute.
	n.ticker15 = time.Tick(minute * 15) //Tick every 15 minutes.
	n.conn = nil
	n.buf = nil
	n.lag = 1000 * 1000 * 1000 * 5 // initial lag of 5 seconds for all irc commands
	n.Disconnected = true
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
	go n.sender()
	go n.pinger()
	go n.ponger()
	go n.ctcp()
	return n
}
