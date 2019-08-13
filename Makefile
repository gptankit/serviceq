# Go params
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=serviceq
BINARY_UNIX_64=$(BINARY_NAME)
BINARY_UNIX_32=$(BINARY_NAME)
BINARY_DARWIN_64=$(BINARY_NAME)
BINARY_DARWIN_32=$(BINARY_NAME)

all: build
build: 
	$(GOBUILD) -o $(BINARY_NAME) -v
	@echo 'done'

build-nodbg:
	$(GOBUILD) -o $(BINARY_NAME) -v -ldflags="-s -w"
	@echo 'done'

test:
	$(GOTEST) -v -race ./... #run tests (TestXxx) excluding benchmarks
	@echo 'done'

bench:
	$(GOTEST) -v -race -run=XXX -bench=. ./... #run all benchmarks (BenchmarkXxx)
	@echo 'done'

test-bench:
	$(GOTEST) -v -race -bench=. ./... #run all tests and benchmarks (TestXxx and BenchmarkXxx)
	@echo 'done'

clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	@echo 'done'

install:
	if [ ! -d /usr/local/serviceq ] ; then \
		sudo mkdir /usr/local/serviceq; \
		sudo mkdir /usr/local/serviceq/config; \
		sudo mkdir /usr/local/serviceq/logs; \
	fi
	sudo cp serviceq /usr/local/serviceq/
	sudo cp sq.properties /usr/local/serviceq/config
	sudo touch /usr/local/serviceq/logs/serviceq_error.log
	sudo rm -f serviceq
	@echo 'Binary location: /usr/local/serviceq/serviceq'
	@echo 'done'

run:
	$(GOBUILD) -o $(BINARY_NAME) -v
	./$(BINARY_NAME)
	@echo 'done'

exec:
	./$(BINARY_NAME)
	@echo 'done'

reload:
	make
	make install
	sudo /opt/serviceq/serviceq

# Cross compilation
build-linux64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX_64) -v
	@echo 'done'
build-linux32:
	CGO_ENABLED=0 GOOS=linux GOARCH=386 $(GOBUILD) -o $(BINARY_UNIX_32) -v
	@echo 'done'
build-mac64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_DARWIN_64) -v
	@echo 'done'
build-mac32:
	CGO_ENABLED=0 GOOS=windows GOARCH=386 $(GOBUILD) -o $(BINARY_DARWIN_32) -v
	@echo 'done'
