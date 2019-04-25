LDFLAGS = -ldflags="-w -s -X github.com/havuz/havuz/cmd.Version=`git describe`"

.PHONY: all build install

all:

build:
	go build $(LDFLAGS)

install:
	go install $(LDFLAGS)
