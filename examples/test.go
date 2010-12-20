package main

import (
	"flag"
	"os"
	"ircchans"
	"log"
	"time"
	"fmt"
	//	"strings"
)

func main() {
	netf := flag.String("net", "viotest.local:6667", "Network name in the form of network.dom:port")
	passf := flag.String("p", "", "Network Password")
	nickf := flag.String("n", "go-irc-chans", "Nickname on network")
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
	log.Println(*netf, *nickf, *userf, *rnf, *passf, *logfile)
	n := ircchans.NewNetwork(*netf, *nickf, *userf, *rnf, *passf, *logfile)
	//test replies, outgoing messages
	/*	go func(){
			chin := make(chan *ircchans.IrcMessage, 100)
			n.RegListener("PRIVMSG", "testreply", chin)
			for !closed(chin) {
				msg := <- chin
				if msg.Params[0] == n.Nick("") {
					n.Privmsg([]string{msg.Prefix}, strings.Join(msg.Params[1:], " ")[1:])
				} else {
					n.Privmsg(msg.Params[:1], strings.Join(msg.Params[1:], " ")[1:])
				}
			}
		}()
	*/
	ticker := time.Tick(1000 * 1000 * 1000 * 15)
	ticker15 := time.Tick(1000 * 1000 * 1000 * 60 * 15)
	for !closed(ticker) {
		select {
		case <-ticker:
			fmt.Println(time.LocalTime())
			if n.Disconnected {
				fmt.Println("Disconnected")
				for err := n.Connect(); err != nil; err = n.Connect() {
					fmt.Printf("Connection failed: %s", err.String())
				}
				n.Join([]string{"#soul9"}, []string{})
			}
		case <-ticker15:
			fmt.Println(n.Whois([]string{n.Nick("")}, ""))
		}
	}
	os.Exit(0)
}
