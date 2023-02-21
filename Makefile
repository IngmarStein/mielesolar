GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=mielesolar

.phony: all test clean update

all: test $(BINARY_NAME)

$(BINARY_NAME): *.go modbus/*.go
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run: $(BINARY_NAME)
	./$(BINARY_NAME)

docker-build: $(BINARY_NAME) Dockerfile
	DOCKER_BUILDKIT=1 docker build .

update:
	go get -u
	GONOPROXY=github.com/ingmarstein/miele-go go get -u github.com/ingmarstein/miele-go@latest
	go mod tidy
