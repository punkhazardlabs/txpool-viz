build:
	go build -o bin/main cmd/main.go

clean:
	rm -rf bin/

test:
	go test ./...

run:
	go run cmd/main.go