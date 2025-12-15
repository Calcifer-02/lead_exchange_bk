ifeq ($(POSTGRES_SETUP),)
	POSTGRES_SETUP := user=postgres password=password dbname=lead_exchange host=localhost port=5432 sslmode=disable
endif

MIGRATION_FOLDER=$(CURDIR)/migrations

OUT_PATH:=$(CURDIR)/pkg
PROTOS_PATH=./api/*.proto
LOCAL_BIN:=$(CURDIR)/bin
BUILD_DIR := ./build

all: bin-deps generate db-up m-create run

db-up:
	docker-compose up

db-down:
	docker-compose down

m-create:
	goose -dir "$(MIGRATION_FOLDER)" create rename_me sql

.migration-up:
	goose -dir "$(MIGRATION_FOLDER)" postgres "$(POSTGRES_SETUP)" up

.migration-down:
	goose -dir "$(MIGRATION_FOLDER)" postgres "$(POSTGRES_SETUP)" down-to 0

up: .migration-up
down: .migration-down
db-reset: .migration-down .migration-up

run:
	go run cmd/main.go

test:
	go test ./...

generate:
	mkdir -p $(OUT_PATH)
	protoc --proto_path api --proto_path vendor.protogen \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go$(EXT) --go_out=${OUT_PATH} --go_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc$(EXT) --go-grpc_out=${OUT_PATH} --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-grpc-gateway=$(LOCAL_BIN)/protoc-gen-grpc-gateway$(EXT) --grpc-gateway_out=${OUT_PATH} --grpc-gateway_opt paths=source_relative \
		--plugin=protoc-gen-openapiv2=$(LOCAL_BIN)/protoc-gen-openapiv2$(EXT) --openapiv2_out=${OUT_PATH} \
		--plugin=protoc-gen-validate=$(LOCAL_BIN)/protoc-gen-validate$(EXT) --validate_out="lang=go,paths=source_relative:${OUT_PATH}" \
		$(PROTOS_PATH)

bin-deps: .vendor-proto
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	GOBIN=$(LOCAL_BIN) go install github.com/envoyproxy/protoc-gen-validate@latest
	GOBIN=$(LOCAL_BIN) go install github.com/rakyll/statik@latest

# ~~~~~~~~~~~~~~
# vendor targets
# ~~~~~~~~~~~~~~
.vendor-proto: .vendor-proto/google/protobuf .vendor-proto/google/api .vendor-proto/validate .vendor-proto/protoc-gen-openapiv2/options

.vendor-proto/protoc-gen-openapiv2/options:
	git clone -b main --single-branch -n --depth=1 --filter=tree:0 \
 		https://github.com/grpc-ecosystem/grpc-gateway vendor.protogen/grpc-ecosystem && \
 		cd vendor.protogen/grpc-ecosystem && \
		git sparse-checkout set --no-cone protoc-gen-openapiv2/options && \
		git checkout
		mkdir -p vendor.protogen/protoc-gen-openapiv2
		mv vendor.protogen/grpc-ecosystem/protoc-gen-openapiv2/options vendor.protogen/protoc-gen-openapiv2
		rm -rf vendor.protogen/grpc-ecosystem

.vendor-proto/google/protobuf:
	git clone -b main --single-branch -n --depth=1 --filter=tree:0 \
	https://github.com/protocolbuffers/protobuf vendor.protogen/protobuf && \
	cd vendor.protogen/protobuf && \
	git sparse-checkout set --no-cone src/google/protobuf &&\
	git checkout
	mkdir -p vendor.protogen/google
	mv vendor.protogen/protobuf/src/google/protobuf vendor.protogen/google
	rm -rf vendor.protogen/protobuf

.vendor-proto/google/api:
	git clone -b master --single-branch -n --depth=1 --filter=tree:0 \
	https://github.com/googleapis/googleapis vendor.protogen/googleapis && \
	cd vendor.protogen/googleapis && \
	git sparse-checkout set --no-cone google/api && \
	git checkout
	mkdir -p vendor.protogen/google
	mv vendor.protogen/googleapis/google/api vendor.protogen/google
	rm -rf vendor.protogen/googleapis

.vendor-proto/validate:
	git clone -b main --single-branch --depth=2 --filter=tree:0 \
	https://github.com/bufbuild/protoc-gen-validate vendor.protogen/tmp && \
	cd vendor.protogen/tmp && \
	git sparse-checkout set --no-cone validate && \
	git checkout
	mkdir -p vendor.protogen/validate
	mv vendor.protogen/tmp/validate vendor.protogen/
	rm -rf vendor.protogen/tmp