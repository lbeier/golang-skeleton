BUILD_DIR ?= build
BINARY=service

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m

.PHONY: all
all: test build

.PHONY: build

build:
	@echo "$(OK_COLOR)==> Building application... $(NO_COLOR)"
	@go build -o ${BUILD_DIR}/${BINARY} ./cmd/service

.PHONY: test
test:
	@echo "$(OK_COLOR)==> Running tests... $(NO_COLOR)"
	@go test -v ./...

.PHONY: clean
clean:
	@echo "$(OK_COLOR)==> Cleaning... $(NO_COLOR)"
	@go clean
	@rm $(BUILD_DIR)/$(BINARY)

.PHONY: run
run: build
	@echo "$(OK_COLOR)==> Running... $(NO_COLOR)"
	./$(BUILD_DIR)/$(BINARY)

.PHONY: deps
deps:
	@echo "$(OK_COLOR)==> Installng dependencies... $(NO_COLOR)"
	@go mod download
