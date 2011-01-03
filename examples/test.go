package main

import (
	"flag"
	"os"
	"github.com/soul9/go-irc-chans" //ircchans
	"log"
	"time"
	"fmt"
	"runtime"
	"strings"
)

const minute = 1000 * 1000 * 1000 * 60

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
	channels := []string{"#soul9"}
	n := ircchans.NewNetwork(*netf, *nickf, *userf, *rnf, *passf, *logfile)
	//test replies, outgoing messages
	go func() {
		chin := make(chan *ircchans.IrcMessage, 100)
		n.Listen.RegListener("PRIVMSG", "testreply", chin)
		for !closed(chin) {
			msg := <-chin
			nick, err := n.Nick("")
			if err == nil && msg.Params[0] == nick && msg.Params[1] == "memusage" {
				targ := strings.Split(msg.Prefix, "!", 2)
				n.Privmsg([]string{targ[0]}, fmt.Sprintf("Currently allocated: %.2fMb, taken from system: %.2fMb.", float(runtime.MemStats.Alloc)/1024/1024, float(runtime.MemStats.Sys)/1024/1024))
			}
		}
		n.Listen.DelListener("PRIVMSG", "testreply")
	}()
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
					time.Sleep(minute / 12)
				}
				if err := n.Join(channels, []string{}); err != nil {
					fmt.Println("Error joining channels %v\n", channels)
					os.Exit(1)
				}
			}
		case <-ticker15:
			nick, _ := n.Nick("")
			fmt.Println(n.Whois([]string{nick}, ""))
		}
	}
	os.Exit(0)
}
