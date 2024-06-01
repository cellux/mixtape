mixtape: $(wildcard *.go) go.mod go.sum
	go build

.PHONY: test
test: mixtape
	./mixtape -f test.tape
