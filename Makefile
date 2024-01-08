all: keylightctl lmutracker

keylightctl: $(wildcard *.go) go.mod go.sum
	go build -ldflags="-w -s -extldflags '-static'" -o keylightctl

lmutracker: lmutracker.mm
	clang -o lmutracker lmutracker.mm -F /Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/System/Library/PrivateFrameworks -framework Foundation -framework IOKit -framework CoreFoundation -framework BezelServices

clean:
	rm -f keylightctl lmutracker

.PHONY: all clean
