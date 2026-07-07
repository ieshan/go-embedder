.PHONY: test fmt vet lint tidy clean

test:
	go test ./...

fmt:
	go fmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	go clean ./...
