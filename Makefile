.PHONY: proto clean install-tools install-protoc download-protos help

SHELL := /bin/bash
PROJECT_DIR := $(CURDIR)
PROJECT_BIN := ${PROJECT_DIR}/bin
PROJECT_INCLUDE := ${PROJECT_DIR}/include
PATH := ${PROJECT_BIN}:${PATH}

PROTOC_GEN_GO_VERSION := v1.28.1
PROTOC_GEN_GO_GRPC_VERSION := v1.2.0
GRPC_GATEWAY_VERSION := v2.15.2
OPENAPIV2_VERSION := v2.15.2
PROTOC_VERSION := 23.4

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

API_OUT_DIR := api/gen
PROTO_FILES := $(shell find api/proto -name "*.proto")

all: proto

# Установка protoc
install-protoc:
ifeq ($(UNAME_S),Linux)
	mkdir -p $(PROJECT_BIN) $(PROJECT_INCLUDE)
	curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip"
	unzip -o "protoc-${PROTOC_VERSION}-linux-x86_64.zip" -d $(PROJECT_DIR)/protoc_temp
	mv $(PROJECT_DIR)/protoc_temp/bin/protoc $(PROJECT_BIN)/
	cp -r $(PROJECT_DIR)/protoc_temp/include/* $(PROJECT_INCLUDE)/
	chmod +x $(PROJECT_BIN)/protoc
	rm -rf $(PROJECT_DIR)/protoc_temp "protoc-${PROTOC_VERSION}-linux-x86_64.zip"
else ifeq ($(UNAME_S),Darwin)
	mkdir -p $(PROJECT_BIN) $(PROJECT_INCLUDE)
ifeq ($(UNAME_M),arm64)
	curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-osx-aarch_64.zip"
	unzip -o "protoc-${PROTOC_VERSION}-osx-aarch_64.zip" -d $(PROJECT_DIR)/protoc_temp
	mv $(PROJECT_DIR)/protoc_temp/bin/protoc $(PROJECT_BIN)/
	cp -r $(PROJECT_DIR)/protoc_temp/include/* $(PROJECT_INCLUDE)/
	chmod +x $(PROJECT_BIN)/protoc
	rm -rf $(PROJECT_DIR)/protoc_temp "protoc-${PROTOC_VERSION}-osx-aarch_64.zip"
else
	curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-osx-x86_64.zip"
	unzip -o "protoc-${PROTOC_VERSION}-osx-x86_64.zip" -d $(PROJECT_DIR)/protoc_temp
	mv $(PROJECT_DIR)/protoc_temp/bin/protoc $(PROJECT_BIN)/
	cp -r $(PROJECT_DIR)/protoc_temp/include/* $(PROJECT_INCLUDE)/
	chmod +x $(PROJECT_BIN)/protoc
	rm -rf $(PROJECT_DIR)/protoc_temp "protoc-${PROTOC_VERSION}-osx-x86_64.zip"
endif
else
	@echo "Unsupported OS: $(UNAME_S). Please install protoc manually."
	@exit 1
endif
	@echo "protoc v${PROTOC_VERSION} installed successfully in $(PROJECT_BIN)"
	@echo "protoc include files copied to $(PROJECT_INCLUDE)"

# Установка зависимостей для генерации кода
install-tools: install-protoc
	mkdir -p $(PROJECT_BIN)
	GOBIN=$(PROJECT_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	GOBIN=$(PROJECT_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	GOBIN=$(PROJECT_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(GRPC_GATEWAY_VERSION)
	GOBIN=$(PROJECT_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(OPENAPIV2_VERSION)

# Загрузка необходимых proto-файлов
download-protos:
	mkdir -p api/third_party/google/api
	mkdir -p api/third_party/protoc-gen-openapiv2/options
	mkdir -p api/third_party/google/protobuf
	curl -sSL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto > api/third_party/google/api/annotations.proto
	curl -sSL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto > api/third_party/google/api/http.proto
	curl -sSL https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/annotations.proto > api/third_party/protoc-gen-openapiv2/options/annotations.proto
	curl -sSL https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/openapiv2.proto > api/third_party/protoc-gen-openapiv2/options/openapiv2.proto

	# Копируем стандартные протобуфные файлы из include директории
	cp -r $(PROJECT_INCLUDE)/google/protobuf/descriptor.proto api/third_party/google/protobuf/ || true
	cp -r $(PROJECT_INCLUDE)/google/protobuf/struct.proto api/third_party/google/protobuf/ || true

	# Загружаем стандартные protobuf файлы напрямую, если они не были скопированы
	if [ ! -f api/third_party/google/protobuf/descriptor.proto ]; then \
		curl -sSL https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf/descriptor.proto > api/third_party/google/protobuf/descriptor.proto; \
	fi
	if [ ! -f api/third_party/google/protobuf/struct.proto ]; then \
		curl -sSL https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf/struct.proto > api/third_party/google/protobuf/struct.proto; \
	fi

# Генерация кода из .proto файлов
proto: clean download-protos
	mkdir -p $(API_OUT_DIR)/go $(API_OUT_DIR)/swagger

	# Генерация Go кода
	PATH="$(PROJECT_BIN):$(PATH)" protoc -I api/proto -I api/third_party -I $(PROJECT_INCLUDE) \
		--go_out=$(API_OUT_DIR)/go --go_opt=paths=source_relative \
		--go-grpc_out=$(API_OUT_DIR)/go --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(API_OUT_DIR)/go --grpc-gateway_opt=paths=source_relative \
		$(PROTO_FILES)

	# Генерация Swagger/OpenAPI спецификации
	PATH="$(PROJECT_BIN):$(PATH)" protoc -I api/proto -I api/third_party -I $(PROJECT_INCLUDE) \
		--openapiv2_out=$(API_OUT_DIR)/swagger --openapiv2_opt=logtostderr=true \
		$(PROTO_FILES)

# Очистка сгенерированных файлов
clean:
	rm -rf $(API_OUT_DIR)

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  make install-protoc  - Установка protoc компилятора"
	@echo "  make install-tools   - Установка инструментов для генерации кода"
	@echo "  make download-protos - Загрузка необходимых proto-файлов"
	@echo "  make proto           - Генерация кода из .proto файлов"
	@echo "  make clean           - Очистка сгенерированных файлов"
	@echo "  make help            - Показать это сообщение с помощью"