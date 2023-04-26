BINARY_NAME=data-proxy

build:
	go mod tidy
	go env -w CGO_ENABLED=0
	go build -o ${BINARY_NAME} -ldflags '-s -w' main.go

run: build
	./${BINARY_NAME}

clean:
	go clean