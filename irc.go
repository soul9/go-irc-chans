package ircchans

import (
	"os"
	"net"
	"log"
	"fmt"
	"strings"
	"bufio"
	"time"
	"sync"
	"encoding/pem"
	"crypto/tls"
	"crypto/rand"
	"crypto/x509"
	"crypto/rsa"
)

const (
	VERSION = "go-irc-chans v0.1"
	minute  = 1000 * 1000 * 1000 * 60
	second  = minute / 60
)

var (
	confdir    = os.Getenv("HOME") + "/.go-irc-chans"
	tlsconfdir = confdir + "/tls"
	certfile   = tlsconfdir + "/clientcert.pem"
	keyfile    = tlsconfdir + "/clientkey.pem"
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
	Listen, OutListen dispatchMap
}

type dispatchMap struct {
	lock  *sync.RWMutex
	chans map[string]map[string]chan *IrcMessage //wildcard * is for any message
}

func CustomTlsConf() (*tls.Config, os.Error) {
	err := os.MkdirAll(tlsconfdir, 0751)
	if err != nil {
		log.Exitf("Couldn't create directory %s: %s", tlsconfdir, err.String())
	}
	confexist := false
	if s, err := os.Stat(certfile); err == nil && s.IsRegular() {
		if s, err := os.Stat(keyfile); err == nil && s.IsRegular() {
			confexist = true
		}
	}
	if !confexist {
		priv, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			return nil, os.NewError(fmt.Sprintf("failed to generate private key: %s", err))
		}
		now := time.Seconds()
		template := x509.Certificate{
			SerialNumber: []byte{0},
			Subject: x509.Name{
				CommonName:   "127.0.0.1",
				Organization: []string{"go-irc-chans"},
			},
			NotBefore: time.SecondsToUTC(now - 300),
			NotAfter:  time.SecondsToUTC(now + 60*60*24*365), // valid for 1 year.

			SubjectKeyId: []byte{1, 2, 3, 4},
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		}
		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		if err != nil {
			return nil, os.NewError(fmt.Sprintf("Failed to create certificate: %s", err.String()))
		}
		certOut, err := os.Open(certfile, os.O_WRONLY|os.O_CREAT, 0644)
		if err != nil {
			return nil, os.NewError(fmt.Sprintf("failed to open %s for writing: %s", certfile, err.String()))
		}
		pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		certOut.Close()
		keyOut, err := os.Open(keyfile, os.O_WRONLY|os.O_CREAT, 0600)
		if err != nil {
			return nil, os.NewError(fmt.Sprintf("failed to open %s for writing:", keyfile, err))
		}
		pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
		keyOut.Close()
	}
	cert, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, os.NewError(fmt.Sprintf("Error reading %s and/or %s for tls config", certfile, keyfile))
	}
	conf := &tls.Config{
		Rand:               rand.Reader,
		Time:               nil,
		Certificates:       []tls.Certificate{cert},
		RootCAs:            nil,
		NextProtos:         nil, // []string{"irc"},
		ServerName:         "",
		AuthenticateClient: true,
		CipherSuites:       nil, //[]uint16{tls.TLS_RSA_WITH_RC4_128_SHA, tls.TLS_RSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA, tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA},
	}
	return conf, nil
}

func (n *Network) Connect() os.Error {
	var err os.Error
	for _, ok := <-n.queueOut; ok; _, ok = <-n.queueOut { //empty the write channel so we don't send out-of-context messages
		continue
	}
	if n.user == "" || n.nick == "" || n.realname == "" {
		return os.NewError("Empty nick and/or user and/or real name")
	}
	tlsConfig, err := CustomTlsConf()
	if err == nil {
		n.conn, err = tls.Dial("tcp", "", n.network, tlsConfig)
	}
	if err != nil {
		n.l.Println("Problem connecting using tls, trying plain-text")
		n.conn, err = net.Dial("tcp", "", n.network)
		if err != nil {
			return os.NewError(fmt.Sprintf("Couldn't connect to network %s: %s.\n", n.network, err.String()))
		}
	}
	n.buf = bufio.NewReadWriter(bufio.NewReader(n.conn), bufio.NewWriter(n.conn))
	n.server = n.conn.RemoteAddr().String()
	n.Disconnected = false
	n.l.Printf("Connected to network %s, server %s\n", n.network, n.server)
	go n.receiver()
	if n.password != "" {
		err = n.Pass()
		if err != nil {
			n.Disconnect("Couldn't authenticate with password")
			return os.NewError("Couldn't register with password")
		}
	}
	_, err = n.Nick(n.nick)
	i := 0
	for err != nil {
		n.nick = fmt.Sprintf("_%s", n.nick)
		_, err = n.Nick(n.nick)
		if i > 5 {
			return os.NewError("Failed to acquire any alternate nick")
		}
		i++
	}
	n.user, err = n.User(n.user)
	if err != nil {
		n.Disconnect("Couldn't register user")
		return os.NewError("Unable to register usename")
	}
	time.Sleep(second) //sleep a second so the lag is more realistic
	n.Ping()
	n.l.Printf("Network lag is: %d nanoseconds", n.lag)
	if n.Disconnected == true {
		return os.NewError("Unknown error, see logs")
	}
	return nil
}

func (n *Network) Reconnect(reason string) os.Error {
	n.l.Printf("Connecting to irc network %s.\n", n.network)
	n.Disconnect(reason)
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
		if n.conn == nil {
			continue
		}
		err, m := PackMsg(msg)
		if err == nil {
			if n.conn == nil || n.buf == nil {
				n.l.Printf("Error writing message (%s): No connection", err.String(), msg)
			}
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
		//dispatch
		go func(msg IrcMessage) {
			n.OutListen.lock.RLock()
			for _, ch := range n.OutListen.chans[msg.Cmd] {
				_ = ch <- &msg
			}
			for _, ch := range n.OutListen.chans["*"] {
				_ = ch <- &msg
			}
			n.OutListen.lock.RUnlock()
			return
		}(m)
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
		go func(msg IrcMessage) {
			n.Listen.lock.RLock()
			for _, ch := range n.Listen.chans[msg.Cmd] {
				_ = ch <- &msg
			}
			for _, ch := range n.Listen.chans["*"] {
				_ = ch <- &msg
			}
			n.Listen.lock.RUnlock()
			return
		}(msg)
	}
	n.l.Println("Something went terribly wrong, receiver exiting")
	return
}

func NewNetwork(net, nick, usr, rn, pass, logfp string) *Network {
	n := new(Network)
	err := os.MkdirAll(confdir, 0751)
	if err != nil {
		log.Exitf("Couldn't create directory %s: %s", confdir, err.String())
	}
	n.network = net
	n.password = pass
	n.nick = nick
	n.user = usr
	n.realname = rn
	n.Listen = dispatchMap{new(sync.RWMutex), make(map[string]map[string]chan *IrcMessage)}
	n.OutListen = dispatchMap{new(sync.RWMutex), make(map[string]map[string]chan *IrcMessage)}
	n.queueOut = make(chan string, 100)
	n.queueIn = make(chan string, 100)
	n.ticker1 = time.Tick(minute)       //Tick every minute.
	n.ticker15 = time.Tick(minute * 15) //Tick every 15 minutes.
	n.conn = nil
	n.buf = nil
	n.lag = timeout(second / 15) // initial lag of 0.5 second for all irc commands
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
	go n.logger()
	return n
}
