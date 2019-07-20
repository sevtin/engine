# by default execute build and install
all: build install

# build the application to check for any compilation errors
build:
	# gofmt -w ./
	# go vet
	go build ./...

# install all dependencies used by the application
deps:
	go get -v -d ./...

# install the application in the Go bin/ folder
install:
	go install ./...

test:
	go test

coverage-test:
	go test gitlab.com/around25/products/matching-engine/engine -coverprofile=coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

# install the application for all architectures targeted
install-all:
	GOOS=linux GOARCH=amd64 go install
	GOOS=darwin GOARCH=amd64 go install
	# GOOS=windows GOARCH=amd64 go install
	# GOOS=windows GOARCH=386 go install