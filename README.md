# Aegis: High-Performance Reverse Proxy

**Author:** Lumir

## Overview

Aegis is a robust, high-performance Layer 7 reverse proxy and load balancer written in GO. Designed for modern, distributed infrastructure, it prioritizes system resilience, traffic shaping, and deep observability. Aegis operates as a central gateway, efficiently distributing traffic across multiple backend instances while protecting the system from overload and backend failures.

## Ethymology
The name 'Aegis' is inspired by the legendary shield carried by Athena and Zeus in Greek mythology, implying that deploying this proxy means operating your infrastructure under divine protection.

## Core Architecture & Features

### 1. Active Health Checks & Auto-Healing (Focus)
System reliability relies on the proxy's ability to detect failing backends before they impact users. Aegis implements an asynchronous, active health-checking engine.
* **Proactive Probing:** Aegis continuously polls the configured /health endpoints of all registered backends at a defined interval.
* **Failure Eviction:** If a backend fails to respond or exceeds the timeout for a consecutive number of times (unhealthy_threshold), Aegis automatically evicts it from the active load-balancing pool.
* **Auto-Recovery:** The proxy continues to probe evicted backends. Once a backend returns successful responses for a set number of consecutive checks (healthy_threshold), it is safely reintroduced into the rotation without dropping traffic.

### 2. Horizontal Scaling & Load Balancing
Aegis facilitates horizontal scaling of your application layer. By defining multiple backends under a single route, Aegis distributes the incoming load.
* **Algorithms:** Supports round_robin for equitable distribution and least_connections to route traffic to the backend currently handling the fewest active requests.
* **Stateless Design:** Aegis itself is stateless. You can deploy multiple Aegis instances behind a DNS round-robin or Layer 4 balancer to scale the proxy layer horizontally.

### 3. HTTPS and TLS Termination
Aegis supports secure communication by acting as a TLS termination proxy. 
* It accepts encrypted HTTPS traffic from the public internet, decrypts it using the provided TLS certificate and private key, and forwards plain HTTP traffic to the internal backends. 
* This offloads CPU-intensive cryptographic operations from your application servers.

### 4. In-Memory Response Caching
To significantly reduce latency and alleviate load on backend servers, Aegis implements an efficient in-memory caching layer. 

* **TTL-Based Storage:** When enabled via the `cache_ttl` configuration directive for a specific route, the proxy stores the HTTP responses in RAM.
* **Backend Offloading:** Subsequent identical requests within the Time-To-Live (TTL) window are served directly from the proxy's memory. This bypasses the backend network call entirely, achieving sub-millisecond response times and protecting the application layer from traffic spikes.


### 5. Hybrid Rate Limiting
To protect the infrastructure from DDoS attacks or abusive clients, Aegis offers a dual-strategy rate limiting system:
* **Standalone Mode (In-Memory):** Uses an efficient Token Bucket algorithm stored in local RAM. Ideal for single-instance deployments.
* **Distributed Mode (Redis-Backed):** When horizontally scaling Aegis itself across multiple servers, the rate limiter utilizes a shared Redis database. This ensures that an IP address cannot bypass quotas by hitting different Aegis proxy instances.

### 6. Observability: Prometheus & Grafana
Understanding traffic patterns is critical. Aegis includes an embedded Prometheus exporter.
* When metrics_enabled is set to true, Aegis exposes a /metrics endpoint.
* It tracks request counts, latency histograms, error rates, and backend status.
* This integrates seamlessly with Grafana, allowing for real-time visualization of the proxy's performance and system health.

### 7. Docker Integration
Aegis is container-native. The entire ecosystem (Proxy, Redis, Prometheus, Grafana, and sample Backends) can be orchestrated using Docker Compose, providing a reproducible environment for development and production.

---

## Getting Started

### Prerequisites
* Go 1.21+
* Docker and Docker Compose (for the full infrastructure stack)

### Testing with Dummy Backends
To quickly test the load balancing and health checking features, you can use Python's built-in HTTP server to simulate backend services. Open two separate terminals and run:

    # Terminal 1: Start Backend 1
    python3 -m http.server 9091

    # Terminal 2: Start Backend 2
    python3 -m http.server 9092

### Running the Full Stack (Recommended)
To evaluate the complete architecture including Redis (for distributed rate limiting) and the Observability stack (Prometheus/Grafana):

1. Clone the repository and navigate to the project root.
2. Start the infrastructure using Docker:
```sh
    docker compose up -d
```
3. Run the Aegis proxy:
```sh
    go run main.go
```

4. Access the proxy at http://localhost:8080 (or https depending on config) and view metrics at http://localhost:3000 (Grafana).

### Running Standalone
If you only need the proxy without external state or metrics:
1. Ensure redis.enabled and observability.metrics_enabled are set to false in config.yaml.
2. Start the proxy: 
    go run main.go

---

## Configuration Reference (config.yaml)

Aegis is configured via a single declarative YAML file. Below is an example demonstrating the routing, HTTPS setup, rate limiting, and health check configurations.

    # Listener configuration defines how Aegis accepts incoming traffic.
    listener:
      port: 8080
      protocol: http # Change to "https" to enable TLS termination
      tls_cert: "./certs/server.crt" # Required if protocol is "https"
      tls_key: "./certs/server.key"  # Required if protocol is "https"
    
    # Enables the /metrics endpoint for Prometheus scraping.
    observability:
      metrics_enabled: true
    
    # Redis configuration for distributed rate limiting.
    redis:
      enabled: true
      addr: "localhost:6379"
      password: ""
      db: 0
    
    # Global network timeouts to mitigate slow-loris attacks.
    defaults:
      timeout_read: 5000
      timeout_write: 5000
      timeout_idle: 30000
    
    # Routing and backend definitions.
    routes:
      - host: "api.local"
        balancer_strategy: "least_connections" 
        rate_limit: 100 # Maximum requests per second per IP
        cache_ttl: 0    # Response cache duration in seconds
        
        backends:
          # Backend 1 with aggressive health checking
          - addr: "http://localhost:9091"
            health_check:
              path: "/health"
              interval: 5000           # Time between checks (ms)
              timeout: 1000            # Max time to wait for a response (ms)
              unhealthy_threshold: 3   # Consecutive failures before eviction
              healthy_threshold: 2     # Consecutive successes before restoration
          
          # Backend 2 configuration
          - addr: "http://localhost:9092"
            health_check:
              path: "/health"
              interval: 5000
              timeout: 1000
              unhealthy_threshold: 3
              healthy_threshold: 2

## Engineering Notes for Reviewers

* **Zero-Allocation Data Path:** Aegis utilizes sync.Pool to recycle byte buffers during request proxying. This architectural decision drastically reduces Garbage Collection (GC) pressure, ensuring flat latency percentiles under heavy concurrent load.
* **Fail-Open Design:** If the distributed Redis instance becomes unreachable, the rate limiting middleware implements a "fail-open" strategy. It logs the connection error and allows traffic to pass, ensuring that an infrastructure failure does not result in a total denial of service for legitimate users.
