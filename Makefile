NAME := tssh
VERSION := $(shell cat ./VERSION)

.PHONY: build
build:
	@make build:macos:arm64
	@make build:macos:amd64
	@make build:linux:arm64
	@make build:linux:amd64

.PHONY: build\:macos
build\:macos:
	@make build:macos:arm64
	@make build:macos:amd64

.PHONY: build\:linux
build\:linux:
	@make build:linux:arm64
	@make build:linux:amd64

.PHONY: build\:macos\:arm64
build\:macos\:arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ${NAME}-${VERSION}-macos-arm64 -ldflags="-s -w -X main.binName=${NAME} -X main.version=${VERSION}" ./cmd

.PHONY: build\:macos\:amd64
build\:macos\:amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ${NAME}-${VERSION}-macos-amd64 -ldflags="-s -w -X main.binName=${NAME} -X main.version=${VERSION}" ./cmd


.PHONY: build\:linux\:arm64
build\:linux\:arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ${NAME}-${VERSION}-linux-arm64 -ldflags="-s -w -X main.binName=${NAME} -X main.version=${VERSION}" ./cmd

.PHONY: build\:linux\:amd64
build\:linux\:amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${NAME}-${VERSION}-linux-amd64 -ldflags="-s -w -X main.binName=${NAME} -X main.version=${VERSION}" ./cmd

.PHONY: help
help:
	@echo "build   				- Compile go code and provide binary for macOS and Linux (arm64 and amd64)"
	@echo "build:macos			- Compile go code and provide binary for macOS (arm64 and amd64)"
	@echo "build:macos:arm64	- Compile go code and provide binary for macOS (arm64)"
	@echo "build:macos:amd64	- Compile go code and provide binary for macOS (amd64)"
	@echo "build:linux			- Compile go code and provide binary for Linux (arm64 and amd64)"
	@echo "build:linux:arm64	- Compile go code and provide binary for Linux (arm64)"
	@echo "build:linux:amd64	- Compile go code and provide binary for Linux (amd64)"