.PHONY: build migrate run tidy

build:
	CGO_ENABLED=0 go build -o cbt-server ./cmd/cbtserver

migrate: build
	./cbt-server migrate

run: build
	./cbt-server

tidy:
	go mod tidy
