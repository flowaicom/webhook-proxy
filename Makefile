all:
	go build -o proxy .

clean:
	rm -f proxy

test:
	go test .
