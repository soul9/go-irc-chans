include $(GOROOT)/src/Make.inc

TARG=ircchans
GOFILES=irc.go ircextras.go bookkeeping.go util.go ctcp.go

include $(GOROOT)/src/Make.pkg
