# Variables
APP_NAME := Goker
SRC_DIR := .
BUILD_DIR := ./bin

# Default target
all: build

# Build the application
build:
	go build -o $(BUILD_DIR)/$(APP_NAME) $(SRC_DIR)

# Run the application
run: build
	$(BUILD_DIR)/$(APP_NAME)

# Clean up build files
clean:
	rm -rf $(BUILD_DIR)

# Phony targets to prevent conflict with files
.PHONY: all build run clean