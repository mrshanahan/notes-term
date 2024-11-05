.DEFAULT_GOAL := build

CMD_DIR = $(CURDIR)/cmd

build:
	go build -o $(CMD_DIR)/notes $(CMD_DIR)/notes.go

install:
	cp -f $(CMD_DIR)/notes ~/bin/notes

run:
	go run $(CMD_DIR)/notes.go

.PHONY: build install run
