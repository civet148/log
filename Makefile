SHELL=/usr/bin/env sh

run:
	cd example && go run main.go
json:
	cd example && go run json_formatter_example.go
clean:
	rm -f *.log; rm -f example/*.log
all: run json

