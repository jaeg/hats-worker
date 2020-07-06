# Go parameters
 GOCMD=go
 GOBUILD=$(GOCMD) build
 GOCLEAN=$(GOCMD) clean
 GOTEST=$(GOCMD) test
 GOGET=$(GOCMD) get
 BINARY_NAME=wart
 BINARY_UNIX=$(BINARY_NAME)_unix
 VERSION=0.1.0

 all: test build
 build:
				 $(GOBUILD) -o ./bin/$(BINARY_NAME) -v
 build-linux:
				CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o ./bin/$(BINARY_UNIX) -v
 test:
				 $(GOTEST) -v ./...
 clean:
				 $(GOCLEAN)
				 rm -f ./bin/$(BINARY_NAME)
 run: build
	./bin/wart --config wart1.config
image: build-linux
	docker build ./ -t jaeg/redis-wart:latest
	docker tag jaeg/redis-wart:latest jaeg/redis-wart:$(VERSION)
publish:
	docker push jaeg/redis-wart:latest
	docker push jaeg/redis-wart:$(VERSION)