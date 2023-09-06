# Makefile

# Go related variables.
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Use the below syntax to set environment variables for the run
# export VAR=value

all: build

# For installing necessary tools for your project
tools:
	go get -u github.com/golang/dep/cmd/dep

# For getting the dependencies of the project
dep:
	dep ensure

# For building the project
build:
	@echo "  >  Building binary..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o $(GOBIN)/my_app $(GOFILES)

# For running the project
run: build
	@echo "  >  Running binary..."
	@$(GOBIN)/my_app

# For cleaning up
clean:
	@echo "  >  Cleaning build cache"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

.PHONY: all tools dep build run clean
