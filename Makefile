include $(GOROOT)/src/Make.inc

TARG=ircchans
GOFILES=irc.go ircextras.go dispatch.go util.go ctcp.go message.go

include $(GOROOT)/src/Make.pkg
