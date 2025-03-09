.PHONY: build start stop logs push

IMAGE_NAME = spinitron-proxy
CONTAINER_NAME = spinitron-proxy-container
NETWORK_NAME = spinitron-proxy-network
APP_PORT = 4001

# Default to Docker, allow override with DOCKER_TOOL=docker or DOCKER_TOOL=podman
DOCKER_TOOL ?= docker

default: clean build run logsf

q: clean build run

exec:
	$(DOCKER_TOOL) exec -it $(CONTAINER_NAME) /bin/bash

logsf:
	$(DOCKER_TOOL) logs -f $(CONTAINER_NAME)

build:
	@echo "Building $(IMAGE_NAME)..."
	@if ! $(DOCKER_TOOL) network ls --format "{{.Name}}" | grep -q "^$(NETWORK_NAME)$$"; then \
		$(DOCKER_TOOL) network create $(NETWORK_NAME); \
	fi
	$(DOCKER_TOOL) build --platform=linux/amd64 --quiet --tag $(IMAGE_NAME) .

start: run

run: stop
	$(DOCKER_TOOL) run --platform=linux/amd64 -d --restart unless-stopped \
		--env SPINITRON_API_KEY=$$SPINITRON_API_KEY \
		-p $(APP_PORT):8080 \
		--network $(NETWORK_NAME) \
		--name $(CONTAINER_NAME) \
		$(IMAGE_NAME)

stop:
	@echo "Checking if container $(CONTAINER_NAME) is running..."
	@if [ "$$($(DOCKER_TOOL) ps -a -q -f name=$(CONTAINER_NAME))" != "" ]; then \
		echo "Stopping $(CONTAINER_NAME)..."; \
		$(DOCKER_TOOL) stop $(CONTAINER_NAME) > /dev/null; \
		echo "Removing the container $(CONTAINER_NAME)..."; \
		$(DOCKER_TOOL) rm -f $(CONTAINER_NAME) > /dev/null; \
	else \
		echo "No running container with name $(CONTAINER_NAME) found."; \
	fi

clean: stop
	@IMAGE_ID=$$($(DOCKER_TOOL) images -q $(IMAGE_NAME)); \
	if [ "$$IMAGE_ID" ]; then \
		echo "Removing image $(IMAGE_NAME) with ID $$IMAGE_ID..."; \
		$(DOCKER_TOOL) rmi $$IMAGE_ID > /dev/null; \
	else \
		echo "No image found with name $(IMAGE_NAME)."; \
	fi
