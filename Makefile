SHELL := /bin/sh
TEMPLFILES := $(shell find . -name '*.templ')
GOFILES  := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: all webui generate

all: webui patchy

generate:
	@echo "=> templ generate"
	~/go/bin/templ generate ./...

webui: generate
	go build -o bin/webui ./cmd/webui

patchy:
	@echo "=> building patchouli"
	go build -o bin/patchy ./main.go

dev:
	@echo "=> starting development server"
	~/go/bin/templ generate --watch --proxy="http://localhost:8080" --cmd="go run ./cmd/webui"

clean:
	@echo "=> clean"
	@find . -name '*_templ.go' -delete