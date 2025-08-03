# Logging Configuration

The kubegraph application uses a structured logging system with configurable log levels.

## Log Levels

The following log levels are supported (in order of increasing severity):

- **DEBUG**: Detailed debug information, useful for troubleshooting
- **INFO**: General information about application operation
- **WARN**: Warning messages for potentially problematic situations
- **ERROR**: Error messages for failed operations

## Configuration

### Environment Variable

Set the `KUBEGRAPH_LOG_LEVEL` environment variable to control logging:

```bash
export KUBEGRAPH_LOG_LEVEL=DEBUG
./kubegraph
```

### Command Line Flag

Use the `--log-level` flag when starting the application:

```bash
./kubegraph --log-level=DEBUG
```

### Default Behavior

If no log level is specified, the application defaults to **INFO** level.

## Examples

### Enable Debug Logging

```bash
# Using environment variable
export KUBEGRAPH_LOG_LEVEL=DEBUG
./kubegraph

# Using command line flag
./kubegraph --log-level=DEBUG
```

### Enable Warning and Error Only

```bash
# Using environment variable
export KUBEGRAPH_LOG_LEVEL=WARN
./kubegraph

# Using command line flag
./kubegraph --log-level=WARN
```

### Production Configuration

For production environments, it's recommended to use INFO or WARN level:

```bash
export KUBEGRAPH_LOG_LEVEL=INFO
./kubegraph
```

## Log Output Format

Log messages are formatted as:

```
[LEVEL] message
```

Example:
```
[INFO] Starting to watch Kubernetes resources...
[DEBUG] Successfully converted to Neo4jDatabase: my-database
[WARN] Resource Neo4jCluster (neo4j.io/v1) not found in cluster, skipping informer setup
[ERROR] Failed to create Neo4j client: connection refused
```

## Usage in Code

The logger can be used in your code as follows:

```go
import "kubegraph/pkg/logger"

// Debug level (only shown when log level is DEBUG)
logger.Debug("Processing object: %s", objectName)

// Info level (shown when log level is INFO or lower)
logger.Info("Successfully processed %d objects", count)

// Warning level (shown when log level is WARN or lower)
logger.Warn("Resource not found, skipping: %s", resourceName)

// Error level (always shown)
logger.Error("Failed to process object: %v", err)
``` 
