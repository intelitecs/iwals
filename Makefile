
main:
	go run cmd/main.go

protobuf:
	protoc --go_out=:api/v1 \
		   --go-grpc_out=require_unimplemented_servers=false:api/v1 \
	       --go_opt=paths=source_relative \
		   --go-grpc_opt=paths=source_relative \
		   --proto_path=:api/v1/protos \
		   api/v1/protos/*.proto


.PHONY: test

test:
	go test -v -race ./test/...
	