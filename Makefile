run: build
	@./bin/nabu

build: test
	@mkdir -p bin
	@go build -o bin/nabu cmd/main.go

release: test
	@mkdir -p release
	@go build -ldflags "-X main.version=v1.0.0" -o release/nabu cmd/main.go
	@cp nabu.json release/
	@cp README.md release/

clean:
	@rm -rf bin release

test:
	@go test ./...

test-pcapreader:
	@go test -v ./test/pcapreader/...

