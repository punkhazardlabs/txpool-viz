BINARY_NAME=txpool-viz
IMAGE_NAME=txpool-viz
ORG_NAME=punkhazardlabs
TAG ?= dev

# Go build for local
build:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME) cmd/main.go

# Clean build artifacts
clean:
	rm -f bin/$(BINARY_NAME)

# Run Go unit tests
test:
	go test ./...

# Tidy Go modules
tidy:
	go mod tidy

# Run the app locally
run-app:
	go run cmd/main.go

# Build frontend assets
build-frontend:
	cd ./frontend && npm install && npm run build --silent

# Build frontend and run app locally
run: build-frontend run-app

# Initialize Docker Buildx
buildx-init:
	docker buildx create --use --name punkhazardlabs-builder || true
	docker buildx inspect --bootstrap

# Multi-arch Docker build and push (linux/amd64 and linux/arm64)
docker-build-push: build build-frontend buildx-init
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(ORG_NAME)/$(IMAGE_NAME):$(TAG) \
		--push .

# Multi-arch Docker build
docker-build: build build-frontend buildx-init
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(ORG_NAME)/$(IMAGE_NAME):$(TAG) \
		--load .
