# Binary and Docker image config
BINARY_NAME=txpool-viz
IMAGE_NAME=txpool-viz
ORG_NAME=punkhazardlabs
TAG ?= dev

# Go build
build:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME) cmd/main.go

# Docker build with tag
docker-build: build
	docker build -t $(ORG_NAME)/$(IMAGE_NAME):$(TAG) .

# Docker run with tag
docker-run:
	docker run --rm $(ORG_NAME)/$(IMAGE_NAME):$(TAG)

# Push image to registry
docker-push:
	docker push $(ORG_NAME)/$(IMAGE_NAME):$(TAG)

# Build and push in one step
docker-build-push: docker-build docker-push

# Clean build artifacts
clean:
	rm -f bin/$(BINARY_NAME)

# Run Go unit tests
test:
	go test ./...

# Build frontend assets (if applicable)
build-frontend:
	cd ./frontend && npm install && npm run build --silent

# Run the app locally
run-app:
	go run cmd/main.go

# Build frontend and run app
run: build-frontend run-app

# Tidy Go modules
tidy:
	go mod tidy
