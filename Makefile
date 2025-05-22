BINARY_NAME=txpool-viz
IMAGE_NAME=txpool-viz
ORG_NAME=punkhazardlabs


build:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME) cmd/main.go

docker-build: build
	docker build -t $(ORG_NAME)/$(IMAGE_NAME) .

docker-run:
	docker run --rm $(ORG_NAME)/$(IMAGE_NAME)

clean:
	rm -f bin/$(BINARY_NAME)

test:
	go test ./...

build-frontend:
	cd ./frontend && npm install && npm run build --silent

run-app:
	go run cmd/main.go

run: build-frontend run-app

tidy:
	go mod tidy
