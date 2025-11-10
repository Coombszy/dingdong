package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/valyala/fasthttp"
)

type Metrics struct {
	totalRequests   atomic.Int64
	totalBodySize   atomic.Int64
	droppedBodies   atomic.Int64
	methodCounts    map[string]*atomic.Int64
	methodCountsMux sync.RWMutex
}

type RequestBody struct {
	body   []byte
	method string
}

var (
	metrics   *Metrics
	bodyQueue chan RequestBody
)

func init() {
	metrics = &Metrics{
		methodCounts: make(map[string]*atomic.Int64),
	}
}

func main() {
	// Customize help message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Ding Dong - High-Performance HTTP Server\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nFeatures:\n")
		fmt.Fprintf(os.Stderr, "  - Request path containing 'dump' will print request details to console\n")
		fmt.Fprintf(os.Stderr, "  - Press Ctrl+C to shutdown and display metrics\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -h 0.0.0.0 -p 8080 -w 50 -q 20000 -b 200\n\n", os.Args[0])
	}

	// Parse command line flags
	host := flag.String("h", "0.0.0.0", "Host to listen on")
	port := flag.Int("p", 61001, "Port to listen on")
	workers := flag.Int("w", 25, "Number of worker goroutines")
	queueSize := flag.Int("q", 10000, "Maximum queue size for body processing")
	maxBodySizeMB := flag.Int("b", 100, "Maximum request body size in MB")
	flag.Parse()

	// Initialize body queue with configured size
	bodyQueue = make(chan RequestBody, *queueSize)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(&wg)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in goroutine
	addr := fmt.Sprintf("%s:%d", *host, *port)
	server := &fasthttp.Server{
		Handler:               requestHandler,
		MaxRequestBodySize:    *maxBodySizeMB * 1024 * 1024, // Convert MB to bytes
		DisableKeepalive:      false,
		ReadBufferSize:        8192,
		WriteBufferSize:       8192,
		Concurrency:           256 * 1024,
		DisableHeaderNamesNormalizing: true,
	}

	go func() {
		log.Printf("Starting server on %s", addr)
		log.Printf("Configuration: workers=%d, queue_size=%d, max_body_size=%dMB", *workers, *queueSize, *maxBodySizeMB)
		if err := server.ListenAndServe(addr); err != nil {
			log.Fatalf("Error in ListenAndServe: %s", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutting down server...")

	// Shutdown server
	if err := server.Shutdown(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Close body queue and wait for workers to finish
	close(bodyQueue)
	wg.Wait()

	// Display metrics
	displayMetrics()
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	// Respond immediately with 200
	ctx.SetStatusCode(fasthttp.StatusOK)

	// Get method and path
	method := string(ctx.Method())
	path := string(ctx.Path())

	// Dump request to console if path contains "dump"
	if strings.Contains(path, "dump") {
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("REQUEST DUMP\n")
		fmt.Printf("Method:      %s\n", method)
		fmt.Printf("Path:        %s\n", path)
		fmt.Printf("Remote Addr: %s\n", ctx.RemoteAddr())
		fmt.Printf("Headers:\n")
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			fmt.Printf("  %s: %s\n", string(key), string(value))
		})
		bodyLen := len(ctx.Request.Body())
		fmt.Printf("Body Size:   %d bytes\n", bodyLen)
		if bodyLen > 0 && bodyLen <= 1024 {
			fmt.Printf("Body:        %s\n", string(ctx.Request.Body()))
		} else if bodyLen > 1024 {
			fmt.Printf("Body:        [%d bytes - too large to display]\n", bodyLen)
		}
		fmt.Println(strings.Repeat("-", 50))
	}

	// Copy body for async processing
	bodyLen := len(ctx.Request.Body())
	if bodyLen > 0 {
		bodyCopy := make([]byte, bodyLen)
		copy(bodyCopy, ctx.Request.Body())

		// Send to worker pool (non-blocking)
		select {
		case bodyQueue <- RequestBody{body: bodyCopy, method: method}:
		default:
			// Queue full, skip this body but still count the request
			metrics.droppedBodies.Add(1)
		}
	}

	// Increment request counter
	metrics.totalRequests.Add(1)
	incrementMethodCount(method)
}

func worker(wg *sync.WaitGroup) {
	defer wg.Done()

	for rb := range bodyQueue {
		// Process body - calculate size and update metrics
		bodySize := int64(len(rb.body))
		metrics.totalBodySize.Add(bodySize)
	}
}

func incrementMethodCount(method string) {
	metrics.methodCountsMux.RLock()
	counter, exists := metrics.methodCounts[method]
	metrics.methodCountsMux.RUnlock()

	if !exists {
		metrics.methodCountsMux.Lock()
		// Double-check after acquiring write lock
		if counter, exists = metrics.methodCounts[method]; !exists {
			counter = &atomic.Int64{}
			metrics.methodCounts[method] = counter
		}
		metrics.methodCountsMux.Unlock()
	}

	counter.Add(1)
}

func displayMetrics() {
	totalReqs := metrics.totalRequests.Load()
	totalBodySize := metrics.totalBodySize.Load()
	droppedBodies := metrics.droppedBodies.Load()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SERVER METRICS")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total Requests:     %d\n", totalReqs)
	fmt.Printf("Total Body Size:    %d bytes (%.2f MB)\n", totalBodySize, float64(totalBodySize)/(1024*1024))

	if totalBodySize > 0 {
		avgBodySize := float64(totalBodySize) / float64(totalReqs) / 1024
		fmt.Printf("Average Body Size:  %.2f KB\n", avgBodySize)
	} else {
		fmt.Printf("Average Body Size:  0 KB (no request bodies)\n")
	}

	fmt.Printf("Dropped Bodies:     %d", droppedBodies)
	if droppedBodies > 0 {
		dropPercentage := float64(droppedBodies) / float64(totalReqs) * 100
		fmt.Printf(" (%.2f%% of requests)", dropPercentage)
	}
	fmt.Println()

	fmt.Println("\nRequests by Method:")
	metrics.methodCountsMux.RLock()
	for method, counter := range metrics.methodCounts {
		count := counter.Load()
		percentage := float64(count) / float64(totalReqs) * 100
		fmt.Printf("  %-8s %d (%.1f%%)\n", method, count, percentage)
	}
	metrics.methodCountsMux.RUnlock()

	fmt.Println(strings.Repeat("=", 50))
}
