GOPATH := ${PWD}:${GOPATH}
export GOPATH

build: ui

install: build-go
	./keys.sh
	cd less/; $(MAKE) install $(MFLAGS)

ui: 
	cd less/; $(MAKE) $(MFLAGS)

build-go:
	go get -d
	go install ./cmd/readas

clean: 
	cd less/; $(MAKE) clean $(MFLAGS)

run: build-go
	./cmd/readas/readas
