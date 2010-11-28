package main

import (
	"fmt"
	"strings"
	"strconv"
	"time"
)
//irc reply message types
var replies = map[string]string{"403": "ERR_NOSUCHCHANNEL",
						"404": "ERR_CANNOTSENDTOCHAN",
						"405": "ERR_TOOMANYCHANNELS",
						"406": "ERR_WASNOSUCHNICK",
						"407": "ERR_TOOMANYTARGETS",
						"409": "ERR_NOORIGIN",
						"411": "ERR_NORECIPIENT",
						"412": "ERR_NOTEXTTOSEND",
						"413": "ERR_NOTOPLEVEL",
						"414": "ERR_WILDTOPLEVEL",
						"421": "ERR_UNKNOWNCOMMAND",
						"422": "ERR_NOMOTD",
						"423": "ERR_NOADMININFO",
						"424": "ERR_FILEERROR",
						"431": "ERR_NONICKNAMEGIVEN",
						"432": "ERR_ERRONEUSNICKNAME",
						"433": "ERR_NICKNAMEINUSE",
						"436": "ERR_NICKCOLLISION",
						"441": "ERR_USERNOTINCHANNEL",
						"442": "ERR_NOTONCHANNEL",
						"443": "ERR_USERONCHANNEL",
						"444": "ERR_NOLOGIN",
						"445": "ERR_SUMMONDISABLED",
						"446": "ERR_USERSDISABLED",
						"451": "ERR_NOTREGISTERED",
						"461": "ERR_NEEDMOREPARAMS",
						"462": "ERR_ALREADYREGISTRED",
						"463": "ERR_NOPERMFORHOST",
						"464": "ERR_PASSWDMISMATCH",
						"465": "ERR_YOUREBANNEDCREEP",
						"467": "ERR_KEYSET",
						"471": "ERR_CHANNELISFULL",
						"472": "ERR_UNKNOWNMODE",
						"473": "ERR_INVITEONLYCHAN",
						"474": "ERR_BANNEDFROMCHAN",
						"475": "ERR_BADCHANNELKEY",
						"481": "ERR_NOPRIVILEGES",
						"482": "ERR_CHANOPRIVSNEEDED",
						"483": "ERR_CANTKILLSERVER",
						"491": "ERR_NOOPERHOST",
						"501": "ERR_UMODEUNKNOWNFLAG",
						"502": "ERR_USERSDONTMATCH",
						"300": "RPL_NONE",
						"302": "RPL_USERHOST",
						"303": "RPL_ISON",
						"301": "RPL_AWAY",
						"305": "RPL_UNAWAY",
						"306": "RPL_NOWAWAY",
						"311": "RPL_WHOISUSER",
						"312": "RPL_WHOISSERVER",
						"313": "RPL_WHOISOPERATOR",
						"317": "RPL_WHOISIDLE",
						"318": "RPL_ENDOFWHOIS",
						"319": "RPL_WHOISCHANNELS",
						"314": "RPL_WHOWASUSER",
						"369": "RPL_ENDOFWHOWAS",
						"321": "RPL_LISTSTART",
						"322": "RPL_LIST",
						"323": "RPL_LISTEND",
						"324": "RPL_CHANNELMODEIS",
						"331": "RPL_NOTOPIC",
						"332": "RPL_TOPIC",
						"341": "RPL_INVITING",
						"342": "RPL_SUMMONING",
						"351": "RPL_VERSION",
						"352": "RPL_WHOREPLY",
						"315": "RPL_ENDOFWHO",
						"353": "RPL_NAMREPLY",
						"366": "RPL_ENDOFNAMES",
						"364": "RPL_LINKS",
						"365": "RPL_ENDOFLINKS",
						"367": "RPL_BANLIST",
						"368": "RPL_ENDOFBANLIST",
						"371": "RPL_INFO",
						"374": "RPL_ENDOFINFO",
						"375": "RPL_MOTDSTART",
						"372": "RPL_MOTD",
						"376": "RPL_ENDOFMOTD",
						"381": "RPL_YOUREOPER",
						"382": "RPL_REHASHING",
						"391": "RPL_TIME",
						"392": "RPL_USERSSTART",
						"393": "RPL_USERS",
						"394": "RPL_ENDOFUSERS",
						"395": "RPL_NOUSERS",
						"200": "RPL_TRACELINK",
						"201": "RPL_TRACECONNECTING",
						"202": "RPL_TRACEHANDSHAKE",
						"203": "RPL_TRACEUNKNOWN",
						"204": "RPL_TRACEOPERATOR",
						"205": "RPL_TRACEUSER",
						"206": "RPL_TRACESERVER",
						"208": "RPL_TRACENEWTYPE",
						"261": "RPL_TRACELOG",
						"211": "RPL_STATSLINKINFO",
						"212": "RPL_STATSCOMMANDS",
						"213": "RPL_STATSCLINE",
						"214": "RPL_STATSNLINE",
						"215": "RPL_STATSILINE",
						"216": "RPL_STATSKLINE",
						"218": "RPL_STATSYLINE",
						"219": "RPL_ENDOFSTATS",
						"241": "RPL_STATSLLINE",
						"242": "RPL_STATSUPTIME",
						"243": "RPL_STATSOLINE",
						"244": "RPL_STATSHLINE",
						"221": "RPL_UMODEIS",
						"251": "RPL_LUSERCLIENT",
						"252": "RPL_LUSEROP",
						"253": "RPL_LUSERUNKNOWN",
						"254": "RPL_LUSERCHANNELS",
						"255": "RPL_LUSERME",
						"256": "RPL_ADMINME",
						"259": "RPL_ADMINEMAIL"}

func (n *Network) Nick(newnick string) string {
	if newnick != "" {
		//TODO: check for correct nick
		n.nick = newnick
		n.queueOut <- fmt.Sprintf("NICK %s", n.nick)
		//TODO: how do we check if the message is successful?
		//replies:
		//ERR_NONICKNAMEGIVEN             ERR_ERRONEUSNICKNAME
		//ERR_NICKNAMEINUSE               ERR_NICKCOLLISION
	}
	return n.nick
}

func (n *Network) User(newuser string) string {
	if n.conn == nil {
		//TODO: can we change the user string once we are connected?
		n.user = newuser
	}
	return n.user
}

func (n *Network) Realname(newrn string) string {
	//TODO: call user from here
	if n.conn == nil {
		//TODO: see User: can we change realname after we are connected? -> if we can change the user after connected
		n.realname = newrn
	}
	return n.realname
}

func (n *Network) NetName(newname string) string{
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

func (n *Network) Join(chans []string, keys []string) {
	for _, ch := range chans {
		if ! strings.HasPrefix(ch, "#") && ! strings.HasPrefix(ch, "&") && ! strings.HasPrefix(ch, "+") && ! strings.HasPrefix(ch, "!") {
			n.l.Printf("Channel %s doesn't start with a legal prefix, returning prematurely", ch)
			return
		}
		if strings.Contains(ch, string(' ')) || strings.Contains(ch, string(7)) || strings.Contains(ch, ",") {
			n.l.Printf("Channel %s contains illegal characters, returning prematurely")
			return
		}
	}
	n.queueOut <- fmt.Sprintf("JOIN %s %s", strings.Join(chans, ","), strings.Join(keys, ","))
	//TODO: replies:
	//ERR_NEEDMOREPARAMS              ERR_BANNEDFROMCHAN
	//ERR_INVITEONLYCHAN              ERR_BADCHANNELKEY
	//ERR_CHANNELISFULL               ERR_BADCHANMASK
	//ERR_NOSUCHCHANNEL               ERR_TOOMANYCHANNELS
	//RPL_TOPIC
	return
}

func (n *Network) Part(chans []string) {
	n.queueOut <- fmt.Sprintf("PART %s", strings.Join(chans, ","))
	//TODO: replies:
	//ERR_NEEDMOREPARAMS              ERR_NOSUCHCHANNEL
	//ERR_NOTONCHANNEL
	return
}

func (n *Network) Mode(target, mode, params string) {
	chmodes := []byte{'o','p','s','i','t','n','m','l','b','v','k'}
	usrmodes := []byte{'i','s','w','o'}
	ok := false
	for _, c := range mode {
		found := false
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
	if !ok {
		for _, c := range mode {
			found := false
			for _, m := range usrmodes {
				if m == byte(c) {
					found = true
					break
				}
			}
			if !found {
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

func (n *Network) Names(chans []string){
	n.queueOut <- fmt.Sprintf("NAMES %s", strings.Join(chans, ","))
	//TODO: replies:
	//RPL_NAMREPLY                    RPL_ENDOFNAMES
	return
}

func (n *Network) List(chans []string, server string){
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

func (n *Network) Invite(target, ch string){
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

func (n *Network) Privmsg(target []string, msg string) {
	n.queueOut <- fmt.Sprintf("PRIVMSG %s :%s", strings.Join(target, ","), msg)
	//TODO: replies:
	//ERR_NORECIPIENT                 ERR_NOTEXTTOSEND
	//ERR_CANNOTSENDTOCHAN            ERR_NOTOPLEVEL
	//ERR_WILDTOPLEVEL                ERR_TOOMANYTARGETS
	//ERR_NOSUCHNICK
	//RPL_AWAY
	return
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

func (n *Network) Whois(target []string, server string) {
	if server == "" {
		n.queueOut <- fmt.Sprintf("WHOIS %s", strings.Join(target, ","))
	} else {
		n.queueOut <- fmt.Sprintf("WHOIS %s %s", server, strings.Join(target, ","))
	}
	//TODO: replies:
	//ERR_NOSUCHSERVER                ERR_NONICKNAMEGIVEN
	//RPL_WHOISUSER                   RPL_WHOISCHANNELS
	//RPL_WHOISCHANNELS               RPL_WHOISSERVER
	//RPL_AWAY                        RPL_WHOISOPERATOR
	//RPL_WHOISIDLE                   ERR_NOSUCHNICK
	//RPL_ENDOFWHOIS
	return
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

func (n *Network) PingNick(nick string){
	n.queueOut <- fmt.Sprintf("PING %s", nick)
	//TODO: replies:
	//ERR_NOORIGIN                    ERR_NOSUCHSERVER
	return
}

func (n *Network) Ping() {
	n.queueOut <- fmt.Sprintf("PING %d", time.Nanoseconds())
	//TODO: numeric replies? PingNick?
	return
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
	if len(users) > 53 {  //maximum number of nicks: 512/9 9 is max length of a nick
		return
	}
	n.queueOut <-  fmt.Sprintf("ISON %s", strings.Join(users, " "))
	//TODO: replies
	//RPL_ISON                ERR_NEEDMOREPARAMS
	return
}
