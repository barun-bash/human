.PHONY: build test install clean

build:
	go build -o test-plugin .

test:
	go test ./... -count=1

install: build
	human plugin install --binary ./test-plugin

clean:
	rm -f test-plugin
