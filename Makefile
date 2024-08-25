.PHONY: help test build run deploy clean

BINARY_NAME = keyboard-backlight-daemon
OUTPUT_DIR  = bin/
GO_FLAGS   ?=
NAME       := keyboard-backlight-daemon
PACKAGE    := github.com/markusressel/$(NAME)
GIT_REV    ?= $(shell git rev-parse --short HEAD)
SOURCE_DATE_EPOCH ?= $(shell date +%s)
DATE       ?= $(shell date -u -d @${SOURCE_DATE_EPOCH} +"%Y-%m-%dT%H:%M:%SZ")
VERSION    ?= 1.1.0

test:   ## Run all tests
	@go clean --testcache && go test -v ./...

build: clean
	go build -o ${OUTPUT_DIR}${BINARY_NAME} main.go

run: build
	./${OUTPUT_DIR}${BINARY_NAME}

daemon-restart:
	sudo systemctl restart keyboard-backlight-daemon

daemon-stop:
	sudo systemctl stop keyboard-backlight-daemon

deploy: build daemon-stop
	mkdir -p /usr/bin/
	sudo cp ./${OUTPUT_DIR}${BINARY_NAME} /usr/bin/${BINARY_NAME}
	sudo systemctl restart keyboard-backlight-daemon

clean:
	go clean
	rm -rf ${OUTPUT_DIR}${BINARY_NAME}