include $(GOROOT)/src/Make.inc

TARG=go-charset.googlecode.com/hg/charset/iconv
CGOFILES=\
	iconv.go\

CGO_LDFLAGS=-liconv
CGO_LDFLAGS+=-L/opt/local/lib
include $(GOROOT)/src/Make.pkg
