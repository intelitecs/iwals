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
	mv *.pem *.csr ${CONFIG_PATH}

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
	