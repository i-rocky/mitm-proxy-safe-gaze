run:
	go run main.go interceptor.go certstorage.go base64image.go

test:
	go test -v ./...

run\:test:
	go run cmd/test/main.go

vet:
	go vet ./...

.PHONY: run test vet run\:test
