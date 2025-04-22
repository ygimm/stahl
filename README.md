# stahl
Prototype of flexible Change Data Capture system

## API for Dynamic Configuration Management

Stahl now provides an API for dynamic configuration management and replicating tables. The API is available via both gRPC and REST (HTTP).

### Generating Code from .proto Files

Use the following commands to generate code from .proto files:

```bash
# Clean previous artifacts (if needed)
rm -rf include bin api/third_party api/gen

# Install tools (protoc and code generators)
make install-tools

# Generate code
make proto
```

The `make install-tools` command installs:
- protoc compiler for generating code from .proto files
- protoc-gen-go for generating Go structures
- protoc-gen-go-grpc for generating gRPC clients and servers
- protoc-gen-grpc-gateway for generating HTTP gateway
- protoc-gen-openapiv2 for generating Swagger/OpenAPI specification

### API Endpoints

The API server runs on the following ports:
- gRPC: 8080
- HTTP: 8081

#### Available endpoints:

- **GET /api/v1/config** - Get current configuration
- **PUT /api/v1/config** - Update full configuration
- **GET /api/v1/tables** - Get list of replicated tables
- **POST /api/v1/tables** - Add a new table for replication
- **DELETE /api/v1/tables/{table_name}** - Remove a table from replication
- **PUT /api/v1/tables/mapping** - Update mapping of tables to Kafka topics
- **PUT /api/v1/kafka** - Update Kafka configuration
- **GET /api/v1/status** - Get current replication status

### Building and Running

```bash
# Update dependencies
go mod tidy

# Build the project
go build -o stahl cmd/stahl/main.go

# Run
./stahl
```

## Swagger UI

After generating code, you can use Swagger UI to interact with the API. API documentation is available in OpenAPI/Swagger format at `api/gen/swagger/stahl.swagger.json`.

To view the API through Swagger UI, you can use one of the following methods:

1. **Local Swagger UI**:
   ```bash
   docker run -p 8082:8080 -e SWAGGER_JSON=/swagger/stahl.swagger.json -v $(pwd)/api/gen/swagger:/swagger swaggerapi/swagger-ui
   ```
   Then open http://localhost:8082 in your browser.

2. **Online Swagger Editor**:
   - Visit https://editor.swagger.io/
   - Upload the file api/gen/swagger/stahl.swagger.json
   - Use the built-in interface to explore the API

## Using gRPC Clients

To interact with the gRPC API, you can use generated Go clients from the `stahl/api/gen/go` package:

```go
import (
    "context"
    apiv1 "stahl/api/gen/go"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // Connect to gRPC server
    conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    // Create client
    client := apiv1.NewConfigurationServiceClient(conn)

    // Call API method
    resp, err := client.GetConfig(context.Background(), &apiv1.GetConfigRequest{})
    // ...
}
```
