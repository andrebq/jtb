.PHONY: test watch tidy

test:
	go test ./...

watch:
	modd

tidy:
	go fmt ./...
	go mod tidy
