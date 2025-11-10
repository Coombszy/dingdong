# Ding Dong

A high-performance HTTP server that responds immediately to all requests and collects metrics.

## Features

- Responds with 200 OK to all HTTP methods (GET, POST, PUT, PATCH, HEAD, DELETE)
- Asynchronous request body processing with worker pool
- Collects metrics: total requests, body sizes, method counts
- Press Ctrl+C to display metrics
- Special feature: paths containing "dump" print request details to console

## Quick Start

Build and run:
```bash
go build -o dingdong
./dingdong
```

The server starts on `0.0.0.0:61001` by default.

## Usage

```bash
./dingdong [options]
```

### Options

- `-h` Host to listen on (default: 0.0.0.0)
- `-p` Port to listen on (default: 61001)
- `-w` Number of worker goroutines (default: 25)
- `-q` Maximum queue size for body processing (default: 10000)
- `-b` Maximum request body size in MB (default: 100)

### Example

```bash
./dingdong -h 0.0.0.0 -p 8080 -w 50 -q 20000 -b 200
```

## Curl Examples

### Basic GET request
```bash
curl http://localhost:61001/hello
```

### POST with JSON body
```bash
curl -X POST http://localhost:61001/api/data \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, Ding Dong!"}'
```

### Dump request details
```bash
curl -X POST http://localhost:61001/dump \
  -H "Content-Type: application/json" \
  -H "X-Custom-Header: test" \
  -d '{"test": "data"}'
```

This will print the full request details to the server console.

### PUT request with large body
```bash
curl -X PUT http://localhost:61001/upload \
  -H "Content-Type: text/plain" \
  --data-binary @largefile.txt
```

## Metrics

Press Ctrl+C to stop the server and view metrics:

```
==================================================
SERVER METRICS
==================================================
Total Requests:     1234
Total Body Size:    5242880 bytes (5.00 MB)
Average Body Size:  4.24 KB
Dropped Bodies:     0

Requests by Method:
  GET      800 (64.8%)
  POST     400 (32.4%)
  PUT      34 (2.8%)
==================================================
```
