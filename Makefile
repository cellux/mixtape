mixtape: $(wildcard *.go) go.mod go.sum assets/prelude.tape
	go build

.PHONY: test
test: mixtape
	./mixtape -f test.tape

.PHONY: clean
clean:
	rm -f mixtape
