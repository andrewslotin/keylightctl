keylightctl: $(wildcard *.go) go.mod go.sum
	go build -ldflags="-w -s -extldflags '-static'" -o keylightctl

.PHONY: clean
clean:
	rm -f keylightctl
