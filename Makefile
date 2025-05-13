build:
	go build -o bin/main cmd/main.go

clean:
	rm -rf bin/

test:
	go test ./...

build-frontend:
	cd ./frontend && npm install && npm run build --silent

run-app:
	go run cmd/main.go

run: build-frontend run-app

tidy:
	go mod tidy
