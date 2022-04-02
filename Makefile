
main:
	go run cmd/main.go

protocmsg:
	protoc --go_out=:api/v1 \
	       --go_opt=paths=source_relative \
		   --proto_path=:api/v1/protos \
		   api/v1/protos/*.proto


.PHONY: test

test:
	go test -v -race ./test/...
	