# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=serviceq
BINARY_UNIX_64=$(BINARY_NAME)_unix_64
BINARY_UNIX_32=$(BINARY_NAME)_unix_32
BINARY_WINDOWS_64=$(BINARY_NAME)_win_64
BINARY_WINDOWS_32=$(BINARY_NAME)_win_32

all: build
build: 
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./... #run tests (TestXxx) excluding benchmarks

bench:
	$(GOTEST) -v -run=XXX -bench=. ./... #run all benchmarks (BenchmarkXxx)

test-bench:
	$(GOTEST) -v -bench=. ./... #run all tests and benchmarks (TestXxx and BenchmarkXxx)

clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

install:
	if [ ! -d /opt/serviceq ] ; then \
		sudo mkdir /opt/serviceq; \
		sudo mkdir /opt/serviceq/config; \
	fi
	sudo cp serviceq /opt/serviceq/
	sudo cp sq.properties /opt/serviceq/config
	sudo rm -f serviceq

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
build-windows64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WINDOWS_64) -v
build-windows32:
	CGO_ENABLED=0 GOOS=windows GOARCH=386 $(GOBUILD) -o $(BINARY_WINDOWS_32) -v

