mixtape: $(wildcard *.go) go.mod go.sum assets/prelude.tape
	go build

.PHONY: test
test: mixtape
	@./runtests.sh

.PHONY: clean
clean:
	rm -f mixtape
