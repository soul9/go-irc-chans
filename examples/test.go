package main

import (
  "flag"
  "os"
  "ircchans"
  "log"
  "time"
  "fmt"
)

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
	log.Println(*netf, *nickf, *userf, *rnf, *passf, *logfile)
	n := ircchans.NewNetwork(*netf, *nickf, *userf, *rnf, *passf, *logfile)
	err := n.Connect()
	if err != nil {
		log.Println(err.String())
		os.Exit(1)
	}
	n.Join([]string{"#soul9", "#ubuntu"}, []string{})
/*	go func(){
		chin := make(chan *IrcMessage, 100)
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
*/
	ticker := time.Tick(1000 * 1000 * 1000 * 60 * 1)
	for !closed(ticker) {
		<- ticker
		fmt.Println(time.LocalTime())
	}
	os.Exit(0)
}
