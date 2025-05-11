FRONTEND_DIR=./frontend

BINARY=txpool-viz

build-frontend:
	cd $(FRONTEND_DIR) && npm install && npm run build

build:
	go build -o bin/main cmd/main.go

clean:
	rm -rf bin/

test:
	go test ./...

run-backend:
	go run ./cmd/main.go

tidy:
	go mod tidy

run: build-frontend
	make run-backend