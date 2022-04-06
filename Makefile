CONFIG_PATH=${HOME}/.iwals
CA_CONFIG_PATH=internal/security/ca


.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: gencert
gencert:
	cfssl gencert -initca ${CA_CONFIG_PATH}/ca-csr.json | cfssljson -bare ca
	cfssl gencert -ca=ca.pem \
	              -ca-key=ca-key.pem \
				  -config=${CA_CONFIG_PATH}/ca-config.json \
				  -profile=server ${CA_CONFIG_PATH}/server-csr.json | cfssljson -bare server

	cfssl gencert -ca=ca.pem \
	              -ca-key=ca-key.pem \
				  -config=${CA_CONFIG_PATH}/ca-config.json \
				  -cn="root" \
				  -profile=client ${CA_CONFIG_PATH}/client-csr.json | cfssljson -bare root-client

	cfssl gencert -ca=ca.pem \
	              -ca-key=ca-key.pem \
				  -config=${CA_CONFIG_PATH}/ca-config.json \
				  -cn="nobody" \
				  -profile=client ${CA_CONFIG_PATH}/client-csr.json | cfssljson -bare  nobody-client		  

	mv *.pem *.csr ${CONFIG_PATH}


$(CONFIG_PATH)/model.conf:
	cp  internal/security/authorization/model.conf $(CONFIG_PATH)/model.conf

$(CONFIG_PATH)/policy.csv:
	cp internal/security/authorization/policy.csv $(CONFIG_PATH)/policy.csv

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

test: $(CONFIG_PATH)/model.conf $(CONFIG_PATH)/policy.csv
	go test -v -race ./test/...
	