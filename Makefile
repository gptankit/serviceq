# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=serviceq
BINARY_UNIX_64=$(BINARY_NAME)_unix_64
BINARY_UNIX_32=$(BINARY_NAME)_unix_32

all: build
build: 
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
run:
	$(GOBUILD) -o $(BINARY_NAME) -v
	./$(BINARY_NAME)

exec:
	./$(BINARY_NAME)

# Cross compilation
build-linux64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX_64) -v
build-linux32:
	CGO_ENABLED=0 GOOS=linux GOARCH=386 $(GOBUILD) -o $(BINARY_UNIX_32) -v
