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
	netf := flag.String("net", "irc.freenode.net", "Network address")
	port := flag.String("port", "6667", "Network port")
	passf := flag.String("pass", "", "Network Password")
	nickf := flag.String("nick", "go-irc-chans", "Nickname on network")
	userf := flag.String("user", "", "Irc user (defaults to nick)")
	rnf := flag.String("realname", "go-ircfs", "Real Name (defaults to nick)")
	chans := flag.String("chans", "#go-nuts", "Channles to join separated by commas (e.g. #foo,#bar; defaults to #go-nuts)")
	logfile := flag.String("logfile", "", "File used for logging (default: stderr)")
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
	log.Println(strings.Join([]string{*netf, *port}, ":"), *nickf, *userf, *rnf, *passf, *logfile)
	channels := strings.Split(*chans, ",", -1)
	n := ircchans.NewNetwork(*netf, *port, *nickf, *userf, *rnf, *passf, *logfile)
	//test replies, outgoing messages
	go func() {
		chin := make(chan *ircchans.IrcMessage, 100)
		n.Listen.RegListener("PRIVMSG", "testreply", chin)
		for !closed(chin) {
			msg := <-chin
			nick := n.GetNick()
			if msg.Destination() == nick {
				switch strings.Join(msg.Params[1:], " ") {
				case "memusage":
					targ := strings.Split(msg.Prefix, "!", 2)
					n.Privmsg([]string{targ[0]}, fmt.Sprintf("Currently allocated: %.2fMb, taken from system: %.2fMb", float32(runtime.MemStats.Alloc)/1024/1024, float32(runtime.MemStats.Sys)/1024/1024))
					n.Privmsg([]string{targ[0]}, fmt.Sprintf("Currently allocated (heap): %.2fMb, taken from system (heap): %.2fMb", float32(runtime.MemStats.HeapAlloc)/1024/1024, float32(runtime.MemStats.HeapSys)/1024/1024))
					n.Privmsg([]string{targ[0]}, fmt.Sprintf("Goroutines currently running: %d", runtime.Goroutines()))
					n.Privmsg([]string{targ[0]}, fmt.Sprintf("Next garbage collection will be when heap reaches %.1f Mb.", float32(runtime.MemStats.NextGC)/1024/1024))
				case "reconnect":
					n.Disconnect("Order")
				}
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
					fmt.Printf("Error joining channels %v\n", channels)
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
