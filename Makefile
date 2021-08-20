.PHONY: test watch tidy

test:
	go test -timeout 300ms ./...

watch:
	modd

tidy:
	go fmt ./...
	go mod tidy
