build:
	@go build -o bin/go-repositories

run: build
	@./bin/go-repositories

