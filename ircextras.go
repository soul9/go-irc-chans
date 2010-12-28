package ircchans

import (
	"os"
	"fmt"
	"strings"
	"strconv"
	"time"
)
//irc reply message types
var replies = map[string]string{
	"ERR_NOSUCHNICK":       "401",
	"ERR_NOSUCHSERVER":     "402",
	"ERR_NOSUCHCHANNEL":    "403",
	"ERR_CANNOTSENDTOCHAN": "404",
	"ERR_TOOMANYCHANNELS":  "405",
	"ERR_WASNOSUCHNICK":    "406",
	"ERR_TOOMANYTARGETS":   "407",
	"ERR_NOORIGIN":         "409",
	"ERR_NORECIPIENT":      "411",
	"ERR_NOTEXTTOSEND":     "412",
	"ERR_NOTOPLEVEL":       "413",
	"ERR_WILDTOPLEVEL":     "414",
	"ERR_UNKNOWNCOMMAND":   "421",
	"ERR_NOMOTD":           "422",
	"ERR_NOADMININFO":      "423",
	"ERR_FILEERROR":        "424",
	"ERR_NONICKNAMEGIVEN":  "431",
	"ERR_ERRONEUSNICKNAME": "432",
	"ERR_NICKNAMEINUSE":    "433",
	"ERR_NICKCOLLISION":    "436",
	"ERR_USERNOTINCHANNEL": "441",
	"ERR_NOTONCHANNEL":     "442",
	"ERR_USERONCHANNEL":    "443",
	"ERR_NOLOGIN":          "444",
	"ERR_SUMMONDISABLED":   "445",
	"ERR_USERSDISABLED":    "446",
	"ERR_NOTREGISTERED":    "451",
	"ERR_NEEDMOREPARAMS":   "461",
	"ERR_ALREADYREGISTRED": "462",
	"ERR_NOPERMFORHOST":    "463",
	"ERR_PASSWDMISMATCH":   "464",
	"ERR_YOUREBANNEDCREEP": "465",
	"ERR_KEYSET":           "467",
	"ERR_CHANNELISFULL":    "471",
	"ERR_UNKNOWNMODE":      "472",
	"ERR_INVITEONLYCHAN":   "473",
	"ERR_BANNEDFROMCHAN":   "474",
	"ERR_BADCHANNELKEY":    "475",
	"ERR_NOPRIVILEGES":     "481",
	"ERR_CHANOPRIVSNEEDED": "482",
	"ERR_CANTKILLSERVER":   "483",
	"ERR_NOOPERHOST":       "491",
	"ERR_UMODEUNKNOWNFLAG": "501",
	"ERR_USERSDONTMATCH":   "502",
	"RPL_NONE":             "300",
	"RPL_USERHOST":         "302",
	"RPL_ISON":             "303",
	"RPL_AWAY":             "301",
	"RPL_UNAWAY":           "305",
	"RPL_NOWAWAY":          "306",
	"RPL_WHOISUSER":        "311",
	"RPL_WHOISSERVER":      "312",
	"RPL_WHOISOPERATOR":    "313",
	"RPL_WHOISIDLE":        "317",
	"RPL_ENDOFWHOIS":       "318",
	"RPL_WHOISCHANNELS":    "319",
	"RPL_WHOWASUSER":       "314",
	"RPL_ENDOFWHOWAS":      "369",
	"RPL_LISTSTART":        "321",
	"RPL_LIST":             "322",
	"RPL_LISTEND":          "323",
	"RPL_CHANNELMODEIS":    "324",
	"RPL_NOTOPIC":          "331",
	"RPL_TOPIC":            "332",
	"RPL_INVITING":         "341",
	"RPL_SUMMONING":        "342",
	"RPL_VERSION":          "351",
	"RPL_WHOREPLY":         "352",
	"RPL_ENDOFWHO":         "315",
	"RPL_NAMREPLY":         "353",
	"RPL_ENDOFNAMES":       "366",
	"RPL_LINKS":            "364",
	"RPL_ENDOFLINKS":       "365",
	"RPL_BANLIST":          "367",
	"RPL_ENDOFBANLIST":     "368",
	"RPL_INFO":             "371",
	"RPL_ENDOFINFO":        "374",
	"RPL_MOTDSTART":        "375",
	"RPL_MOTD":             "372",
	"RPL_ENDOFMOTD":        "376",
	"RPL_YOUREOPER":        "381",
	"RPL_REHASHING":        "382",
	"RPL_TIME":             "391",
	"RPL_USERSSTART":       "392",
	"RPL_USERS":            "393",
	"RPL_ENDOFUSERS":       "394",
	"RPL_NOUSERS":          "395",
	"RPL_TRACELINK":        "200",
	"RPL_TRACECONNECTING":  "201",
	"RPL_TRACEHANDSHAKE":   "202",
	"RPL_TRACEUNKNOWN":     "203",
	"RPL_TRACEOPERATOR":    "204",
	"RPL_TRACEUSER":        "205",
	"RPL_TRACESERVER":      "206",
	"RPL_TRACENEWTYPE":     "208",
	"RPL_TRACELOG":         "261",
	"RPL_STATSLINKINFO":    "211",
	"RPL_STATSCOMMANDS":    "212",
	"RPL_STATSCLINE":       "213",
	"RPL_STATSNLINE":       "214",
	"RPL_STATSILINE":       "215",
	"RPL_STATSKLINE":       "216",
	"RPL_STATSYLINE":       "218",
	"RPL_ENDOFSTATS":       "219",
	"RPL_STATSLLINE":       "241",
	"RPL_STATSUPTIME":      "242",
	"RPL_STATSOLINE":       "243",
	"RPL_STATSHLINE":       "244",
	"RPL_UMODEIS":          "221",
	"RPL_LUSERCLIENT":      "251",
	"RPL_LUSEROP":          "252",
	"RPL_LUSERUNKNOWN":     "253",
	"RPL_LUSERCHANNELS":    "254",
	"RPL_LUSERME":          "255",
	"RPL_ADMINME":          "256",
	"RPL_ADMINEMAIL":       "259"}

func timeout(lag int64) int64 {
	return lag + (second)
}

func (n *Network) Pass() os.Error {
	t := strconv.Itoa64(time.Nanoseconds())
	myreplies := []string{"ERR_NEEDMOREPARAMS", "ERR_ALREADYREGISTRED"}
	var err os.Error
	repch := make(chan *IrcMessage)
	for _, rep := range myreplies {
		if err := n.RegListener(replies[rep], t, repch); err != nil {
			err = os.NewError(fmt.Sprintf("Couldn't authenticate with password, exiting: %s", err.String()))
		}
	}
	ticker := time.NewTicker(timeout(n.lag)) //timeout in lag + 5 seconds
	defer func(myreplies []string, t string, tick *time.Ticker) {
		for _, rep := range myreplies {
			n.DelListener(replies[rep], t)
		}
		tick.Stop()
		return
	}(myreplies, t, ticker)
	n.queueOut <- fmt.Sprintf("PASS %s", n.password)
	select {
	case msg := <-repch:
		if msg.Cmd == replies["ERR_NEEDMOREPARAMS"] {
			err = os.NewError(fmt.Sprintf("Need more parameters for password: %s", msg.String()))
		}
		break
	case <-ticker.C:
		break
	}
	return err
}

func (n *Network) Nick(newnick string) (string, os.Error) {
	t := strconv.Itoa64(time.Nanoseconds())
	ticker := time.NewTicker(timeout(n.lag)) //timeout in lag+5 seconds
	myreplies := []string{"ERR_NONICKNAMEGIVEN", "ERR_ERRONEUSNICKNAME", "ERR_NICKNAMEINUSE", "ERR_NICKCOLLISION"}
	if newnick == "" {
		return n.nick, nil
	}
	//TODO: check for correct nick (illegal characters)
	if len(newnick) > 9 {
		newnick = newnick[:9]
	}
	repch := make(chan *IrcMessage)
	for _, rep := range myreplies {
		if err := n.RegListener(replies[rep], t, repch); err != nil {
			for _, rep := range myreplies {
				n.DelListener(replies[rep], t)
			}
			return n.nick, os.NewError("Unable to register new listener")
		}
	}
	defer func(myreplies []string, t string, tick *time.Ticker) {
		for _, rep := range myreplies {
			n.DelListener(replies[rep], t)
		}
		tick.Stop()
		return
	}(myreplies, t, ticker)
	n.queueOut <- fmt.Sprintf("NICK %s", newnick)
	select {
	case msg := <-repch:
		if msg.Cmd == replies["ERR_ERRONEUSNICKNAME"] || msg.Cmd == replies["ERR_NICKNAMEINUSE"] || msg.Cmd == replies["ERR_NICKCOLLISION"] {
			for key, _ := range replies {
				if replies[key] == msg.Cmd {
					return n.nick, os.NewError(key)
				}
			}
			return n.nick, os.NewError("Unknown error")
		}
	case <-ticker.C:
		break
	}
	n.nick = newnick
	return n.nick, nil
}


func (n *Network) User(newuser string) (string, os.Error) {
	t := strconv.Itoa64(time.Nanoseconds())
	ticker := time.NewTicker(timeout(n.lag)) //timeout in lag+5 seconds
	myreplies := []string{"ERR_NEEDMOREPARAMS", "ERR_ALREADYREGISTRED"}
	if newuser == "" {
		return n.user, nil
	} else if len(newuser) > 9 {
		newuser = newuser[:9]
	}
	repch := make(chan *IrcMessage)
	for _, rep := range myreplies {
		if err := n.RegListener(replies[rep], t, repch); err != nil {
			return "", os.NewError(fmt.Sprintf("Couldn't register Listener for %s: %s", replies[rep], err.String()))
		}
	}
	defer func(myreplies []string, t string, tick *time.Ticker) {
		for _, rep := range myreplies {
			n.DelListener(replies[rep], t)
		}
		tick.Stop()
		return
	}(myreplies, t, ticker)
	n.queueOut <- fmt.Sprintf("USER %s 0.0.0.0 0.0.0.0 :%s", n.user, n.realname)
	select {
	case msg := <-repch:
		if msg.Cmd == replies["ERR_NEEDMOREPARAMS"] {
			return n.user, os.NewError("ERR_NEEDMOREPARAMS")
		} else if msg.Cmd == replies["ERR_ALREADYREGISTRED"] {
			return n.user, os.NewError("ERR_ALREADYREGISTRED")
		}
	case <-ticker.C:
		n.user = newuser
	}
	return n.user, nil
}

func (n *Network) Realname(newrn string) string {
	//TODO: call user from here
	if n.conn == nil {
		//TODO: see User: can we change realname after we are connected? -> if we can change the user after connected
		n.realname = newrn
	}
	return n.realname
}

func (n *Network) NetName(newname string) string {
	if newname != "" {
		n.network = newname
		//TODO: reconnect
	}
	return n.network
}

func (n *Network) SysOpMe(user, pass string) {
	n.queueOut <- fmt.Sprintf("OPER %s %s", user, pass)
	//TODO: replies:
	//ERR_NEEDMOREPARAMS              RPL_YOUREOPER
	//ERR_NOOPERHOST                  ERR_PASSWDMISMATCH
	return
}

func (n *Network) Quit(reason string) {
	n.queueOut <- fmt.Sprintf("QUIT :%s", reason)
	return
}

func (n *Network) Join(chans []string, keys []string) os.Error {
	if len(chans) == 0 {
		return os.NewError("No channels given")
	}
	t := strconv.Itoa64(time.Nanoseconds())
	ticker := time.NewTicker(timeout(n.lag)) //timeout in lag+5 seconds
	myreplies := []string{"ERR_NEEDMOREPARAMS", "ERR_BANNEDFROMCHAN",
		"ERR_INVITEONLYCHAN", "ERR_BADCHANNELKEY",
		"ERR_CHANNELISFULL", "ERR_BADCHANMASK",
		"ERR_NOSUCHCHANNEL", "ERR_TOOMANYCHANNELS",
		"RPL_TOPIC", "JOIN"}
	for _, ch := range chans {
		if !strings.HasPrefix(ch, "#") && !strings.HasPrefix(ch, "&") && !strings.HasPrefix(ch, "+") && !strings.HasPrefix(ch, "!") {
			return os.NewError(fmt.Sprintf("Channel %s doesn't start with a legal prefix", ch))
		}
		if strings.Contains(ch, string(' ')) || strings.Contains(ch, string(7)) || strings.Contains(ch, ",") {
			return os.NewError(fmt.Sprintf("Channel %s contains illegal characters", ch))
		}
	}
	repch := make(chan *IrcMessage)
	defer func(myreplies []string, t string) {
		for _, rep := range myreplies {
			_, ok := replies[rep]
			if ok {
				n.DelListener(replies[rep], t)
			} else {
				n.DelListener(rep, t)
			}
		}
		return
	}(myreplies, t)
	for _, rep := range myreplies {
		_, ok := replies[rep]
		if ok {
			if err := n.RegListener(replies[rep], t, repch); err != nil {
				return os.NewError(fmt.Sprintf("Couldn't register listener %s: %s", replies[rep], err.String()))
			}
		} else {
			if err := n.RegListener(rep, t, repch); err != nil {
				return os.NewError(fmt.Sprintf("Couldn't register listener %s: %s", rep, err.String()))
			}
		}
	}
	n.queueOut <- fmt.Sprintf("JOIN %s %s", strings.Join(chans, ","), strings.Join(keys, ","))
	joined := 0
	for {
		select {
		case msg := <-repch:
			if msg.Cmd == "JOIN" {
				for _, chn := range chans {
					if msg.Params[0] == fmt.Sprintf(":%s", chn) {
						joined++
						break
					}
				}
			} else {
				for key, _ := range replies {
					if replies[key] == msg.Cmd {
						if key[:3] == "ERR" {
							return os.NewError(key)
						}
					}
				}
			}
			if joined == len(chans) {
				return nil
			}
			ticker.Stop()
			ticker = time.NewTicker(timeout(n.lag))
		case <-ticker.C:
			return os.NewError("Didn't receive join reply")
		}
	}
	ticker.Stop()
	return nil
}

func (n *Network) Part(chans []string, reason string) {
	n.queueOut <- fmt.Sprintf("PART %s :%s", strings.Join(chans, ","), reason)
	//TODO: replies:
	//ERR_NEEDMOREPARAMS              ERR_NOSUCHCHANNEL
	//ERR_NOTONCHANNEL
	return
}

func (n *Network) Mode(target, mode, params string) {
	chmodes := []byte{'o', 'p', 's', 'i', 't', 'n', 'm', 'l', 'b', 'v', 'k'}
	usrmodes := []byte{'i', 's', 'w', 'o'}
	var found bool
	for _, c := range mode { //is it a channel mode?
		found = false
		for _, m := range chmodes {
			if m == byte(c) {
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	if !found { //maybe it's a user mode?
		for _, c := range mode {
			found := false
			for _, m := range usrmodes {
				if m == byte(c) {
					found = true
					break
				}
			}
			if !found { //neither a channel nor a user mode, don't touch this
				return
				//TODO: return error?
			}
		}
	}
	n.queueOut <- fmt.Sprintf("MODE %s %s %s", target, mode, params)
	//TODO: replies:
	//ERR_NEEDMOREPARAMS              RPL_CHANNELMODEIS
	//ERR_CHANOPRIVSNEEDED            ERR_NOSUCHNICK
	//ERR_NOTONCHANNEL                ERR_KEYSET
	//RPL_BANLIST                     RPL_ENDOFBANLIST
	//ERR_UNKNOWNMODE                 ERR_NOSUCHCHANNEL
	//
	//ERR_USERSDONTMATCH              RPL_UMODEIS
	//ERR_UMODEUNKNOWNFLAG
	return
}

func (n *Network) Topic(ch, topic string) {
	if topic == "" {
		n.queueOut <- fmt.Sprintf("TOPIC %s", ch)
	} else {
		n.queueOut <- fmt.Sprintf("TOPIC %s :%s", ch, topic)
	}
	//TODO: replies
	//ERR_NEEDMOREPARAMS              ERR_NOTONCHANNEL
	//RPL_NOTOPIC                     RPL_TOPIC
	//ERR_CHANOPRIVSNEEDED
	return
}

func (n *Network) Names(chans []string) {
	n.queueOut <- fmt.Sprintf("NAMES %s", strings.Join(chans, ","))
	//TODO: replies:
	//RPL_NAMREPLY                    RPL_ENDOFNAMES
	return
}

func (n *Network) List(chans []string, server string) {
	raw := "LIST"
	if len(chans) > 0 {
		raw += fmt.Sprintf(" %s", strings.Join(chans, ","))
	}
	if server != "" {
		raw += fmt.Sprintf(" %s", server)
	}
	n.queueOut <- raw
	//TODO: replies:
	//ERR_NOSUCHSERVER                RPL_LISTSTART
	//RPL_LIST                        RPL_LISTEND
	return
}

func (n *Network) Invite(target, ch string) {
	n.queueOut <- fmt.Sprintf("INVITE %s %s", target, ch)
	//TODO: replies:
	//ERR_NEEDMOREPARAMS              ERR_NOSUCHNICK
	//ERR_NOTONCHANNEL                ERR_USERONCHANNEL
	//ERR_CHANOPRIVSNEEDED
	//RPL_INVITING                    RPL_AWAY
	return
}

func (n *Network) Kick(ch, target, reason string) {
	if reason == "" {
		n.queueOut <- fmt.Sprintf("KICK %s %s", ch, target)
	} else {
		n.queueOut <- fmt.Sprintf("KICK %s %s :%s", ch, target, reason)
	}
	//TODO: replies:
	//ERR_NEEDMOREPARAMS              ERR_NOSUCHCHANNEL
	//ERR_BADCHANMASK                 ERR_CHANOPRIVSNEEDED
	//ERR_NOTONCHANNEL
	return
}

func (n *Network) Privmsg(target []string, msg string) os.Error { //BUG: make privmsg hack up messages that are too long
	t := strconv.Itoa64(time.Nanoseconds())
	ticker := time.NewTicker(timeout(n.lag)) //timeout in lag+5 seconds
	myreplies := []string{"ERR_NORECIPIENT", "ERR_NOTEXTTOSEND",
		"ERR_CANNOTSENDTOCHAN", "ERR_NOTOPLEVEL",
		"ERR_WILDTOPLEVEL", "ERR_TOOMANYTARGETS",
		"ERR_NOSUCHNICK", "RPL_AWAY"}
	repch := make(chan *IrcMessage)
	for _, rep := range myreplies {
		if err := n.RegListener(replies[rep], t, repch); err != nil {
			return os.NewError(fmt.Sprintf("Couldn't register nick %s: %s", replies[rep], err.String()))
		}
	}
	defer func(myreplies []string, t string) {
		for _, rep := range myreplies {
			n.DelListener(replies[rep], t)
		}
		return
	}(myreplies, t)
	n.queueOut <- fmt.Sprintf("PRIVMSG %s :%s", strings.Join(target, ","), msg)
	for {
		select {
		case msg := <-repch:
			for key, _ := range replies {
				if replies[key] == msg.Cmd && key[:3] == "ERR" {
					return os.NewError(key)
				}
			}
			ticker.Stop()
			ticker = time.NewTicker(timeout(n.lag))
		case <-ticker.C:
			ticker.Stop()
			return nil
		}
	}
	ticker.Stop()
	return nil
}

func (n *Network) Notice(target, text string) {
	n.queueOut <- fmt.Sprintf("NOTICE %s :%s", target, text)
	//TODO: replies:
	//ERR_NORECIPIENT                 ERR_NOTEXTTOSEND
	//ERR_CANNOTSENDTOCHAN            ERR_NOTOPLEVEL
	//ERR_WILDTOPLEVEL                ERR_TOOMANYTARGETS
	//ERR_NOSUCHNICK
	//RPL_AWAY
	return
}

func (n *Network) Who(target string) {
	n.queueOut <- fmt.Sprintf("WHO %s", target)
	//TODO: replies:
	//ERR_NOSUCHSERVER
	//RPL_WHOREPLY                    RPL_ENDOFWHO
	return
}

func (n *Network) Whois(target []string, server string) map[string][]string { //TODO: return a map[string][][]string? map[string][]IrcMessage?
	t := strconv.Itoa64(time.Nanoseconds())
	ticker := time.NewTicker(timeout(n.lag)) //timeout after lag+5 seconds
	myreplies := []string{"ERR_NOSUCHSERVER", "ERR_NONICKNAMEGIVEN",
		"RPL_WHOISUSER", "RPL_WHOISCHANNELS",
		"RPL_WHOISSERVER", "RPL_AWAY",
		"RPL_WHOISOPERATOR", "RPL_WHOISIDLE",
		"ERR_NOSUCHNICK", "RPL_ENDOFWHOIS"}
	repch := make(chan *IrcMessage)
	for _, rep := range myreplies {
		if err := n.RegListener(replies[rep], t, repch); err != nil {
			n.l.Printf("Couldn't whois %d=%s: %s", replies[rep], rep, err.String())
			os.Exit(1)
		}
	}
	defer func(myreplies []string, t string) {
		for _, rep := range myreplies {
			n.DelListener(replies[rep], t)
		}
		return
	}(myreplies, t)

	if server == "" {
		n.queueOut <- fmt.Sprintf("WHOIS %s", strings.Join(target, ","))
	} else {
		n.queueOut <- fmt.Sprintf("WHOIS %s %s", server, strings.Join(target, ","))
	}
	info := make(map[string][]string)
	for _, rep := range myreplies {
		info[replies[rep]] = make([]string, 0)
	}
	for {
		select {
		case m := <-repch:
			info[m.Cmd] = append(info[m.Cmd], strings.Join((*m).Params, " "))
			if m.Cmd == replies["RPL_ENDOFWHOIS"] {
				return info
			}
			ticker.Stop()
			ticker = time.NewTicker(timeout(n.lag)) //restart the ticker to timeout correctly
		case <-ticker.C:
			ticker.Stop()
			return info
		}
	}
	ticker.Stop()
	return info //BUG: why do we need this?
}

func (n *Network) Whowas(target string, count int, server string) {
	var raw string
	if server != "" {
		raw = fmt.Sprintf("WHOIS %s ", server, target)
	}
	raw += fmt.Sprintf("%s %s", target, strconv.Itoa(count))
	n.queueOut <- raw
	//TODO: replies:
	//ERR_NONICKNAMEGIVEN             ERR_WASNOSUCHNICK
	//RPL_WHOWASUSER                  RPL_WHOISSERVER
	//RPL_ENDOFWHOWAS
	return
}

func (n *Network) PingNick(nick string) {
	n.queueOut <- fmt.Sprintf("PING %s", nick)
	//TODO: replies:
	//ERR_NOORIGIN                    ERR_NOSUCHSERVER
	return
}

func (n *Network) Ping() (int64, os.Error) {
	myreplies := []string{"ERR_NOORIGIN", "ERR_NOSUCHSERVER"}
	t := strconv.Itoa64(time.Nanoseconds())
	repch := make(chan *IrcMessage)
	ticker := time.NewTicker(timeout(n.lag))
	for _, rep := range myreplies {
		n.RegListener(replies[rep], t, repch)
	}
	n.RegListener("PONG", t, repch)
	var rep *IrcMessage
	defer func(myreplies []string, t string, n *Network, tick *time.Ticker) {
		for _, rep := range myreplies {
			n.DelListener(replies[rep], t)
		}
		n.DelListener("PONG", t)
		tick.Stop()
		return
	}(myreplies, t, n, ticker)
	n.queueOut <- fmt.Sprintf("PING %d", time.Nanoseconds())
	select {
	case <-ticker.C:
		return 0, os.NewError("Timeout in receiving reply")
	case rep = <-repch:
	}
	if rep.Cmd == "PONG" {
		origtime, err := strconv.Atoi64(rep.Params[len(rep.Params)-1][1:])
		if err == nil {
			n.lag = time.Nanoseconds() - origtime
			return time.Nanoseconds() - origtime, err
		} else {
			return 0, err
		}
	} else {
		switch rep.Cmd {
		case replies["ERR_NOORIGIN"]:
			return 0, os.NewError("ERR_NOORIGIN")
		case replies["ERR_NOSUCHSERVER"]:
			return 0, os.NewError("ERR_NOSUCHSERVER")
		default:
			return 0, os.NewError("Unknown error")
		}
	}
	return 0, os.NewError("Unknown error")
}

func (n *Network) Pong(msg string) {
	n.queueOut <- fmt.Sprintf("PONG %s", msg)
	//TODO: numeric replies? PingNick?
	return
}

func (n *Network) Away(reason string) {
	raw := fmt.Sprintf("AWAY")
	if reason != "" {
		raw += fmt.Sprintf(" :%s", reason)
	}
	raw += ""
	n.queueOut <- raw
	//TODO: replies:
	//RPL_UNAWAY                      RPL_NOWAWAY
	return
}

func (n *Network) Users(server string) {
	raw := fmt.Sprintf("USERS")
	if server != "" {
		raw += fmt.Sprintf(" %s", server)
	}
	raw += ""
	n.queueOut <- raw
	return
}

func (n *Network) Userhost(users []string) {
	if len(users) > 5 {
		//todo cycle them 5-by-5?
		return
	}
	n.queueOut <- fmt.Sprintf("USERHOST %s", strings.Join(users, " "))
	//TODO: replies
	//RPL_USERHOST                    ERR_NEEDMOREPARAMS
	return
}

func (n *Network) Ison(users []string) {
	if len(users) > 53 { //maximum number of nicks: 512/9 9 is max length of a nick
		return
	}
	n.queueOut <- fmt.Sprintf("ISON %s", strings.Join(users, " "))
	//TODO: replies
	//RPL_ISON                ERR_NEEDMOREPARAMS
	return
}
