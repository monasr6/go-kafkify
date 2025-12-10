# Load Testing with K6

This directory contains K6 load testing scripts for the Go-Kafkify microservices platform.

## Prerequisites

Install K6:
```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Windows
choco install k6
```

## Test Scripts

### 1. REST API Load Test (`rest-api-test.js`)

Tests the REST service with realistic user scenarios including:
- Creating resources
- Retrieving resources
- Listing all resources
- Updating resources
- Deleting resources
- Health checks

**Run the test:**
```bash
k6 run load-tests/rest-api-test.js

# With custom base URL
k6 run -e BASE_URL=http://your-service:8080 load-tests/rest-api-test.js
```

**Test stages:**
1. Ramp up to 20 users (30s)
2. Stay at 50 users (1m)
3. Ramp up to 100 users (30s)
4. Stay at 100 users (2m)
5. Ramp down to 0 (30s)

**Success criteria:**
- 95% of requests < 500ms
- Error rate < 10%
- Failed requests < 5%

### 2. Kafka Throughput Test (`kafka-throughput-test.js`)

Tests Kafka event generation throughput by creating high volumes of resources:
- Batch resource creation
- Multiple operations per iteration
- Measures events generated per second

**Run the test:**
```bash
k6 run load-tests/kafka-throughput-test.js

# With custom configuration
k6 run -e BASE_URL=http://your-service:8080 load-tests/kafka-throughput-test.js
```

**Test stages:**
1. Ramp up to 50 VUs (30s)
2. Ramp up to 100 VUs (1m)
3. Ramp up to 200 VUs (2m)
4. Hold at 200 VUs (1m)
5. Ramp down to 0 (30s)

**Success criteria:**
- 95% of requests < 1000ms
- Failed requests < 5%
- At least 1000 resources created

## Interpreting Results

### Key Metrics

**HTTP Metrics:**
- `http_req_duration` - Request duration times
- `http_req_failed` - Percentage of failed requests
- `http_reqs` - Total number of HTTP requests

**Custom Metrics:**
- `resources_created` - Total resources created
- `events_generated` - Total Kafka events generated
- `resource_creation_time` - Time to create a resource
- `resource_retrieval_time` - Time to retrieve a resource

### Understanding Throughput

The Kafka throughput test measures:
1. **HTTP throughput**: Requests per second to the REST API
2. **Event throughput**: Kafka messages per second (via Outbox pattern)
3. **End-to-end latency**: Time from HTTP request to event in outbox table

Each resource operation generates one Kafka event:
- Create → `resource.created`
- Update → `resource.updated`
- Delete → `resource.deleted`

### Expected Performance

On a typical 3-node Kubernetes cluster:

**REST API:**
- Throughput: ~10,000 req/s
- P95 Latency: < 50ms
- P99 Latency: < 100ms

**Kafka Events:**
- Throughput: ~100,000 msg/s
- End-to-end latency: < 100ms

## Monitoring During Tests

While running load tests, monitor:

1. **Grafana Dashboards**: http://localhost:3000
   - Service latency
   - Request rates
   - Error rates
   - Resource utilization

2. **Prometheus Metrics**: http://localhost:9091
   - Query specific metrics
   - View time series data

3. **Service Logs**:
```bash
# Docker Compose
docker-compose logs -f rest-service

# Kubernetes
kubectl logs -f -n go-kafkify -l app=rest-service
```

## Advanced Usage

### Custom Scenarios

Create custom test scenarios by modifying the `options` object:

```javascript
export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-vus',
      vus: 50,
      duration: '5m',
    },
    spike_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '1m', target: 100 },
        { duration: '10s', target: 1000 }, // Spike!
        { duration: '3m', target: 1000 },
        { duration: '10s', target: 100 },
      ],
    },
  },
};
```

### Running Against Kubernetes

```bash
# Port-forward the REST service
kubectl port-forward -n go-kafkify svc/rest-service 8080:8080

# Run test
k6 run load-tests/rest-api-test.js
```

### Cloud Testing with K6 Cloud

```bash
# Sign up at https://k6.io/cloud/
k6 login cloud

# Run test in the cloud
k6 cloud load-tests/rest-api-test.js
```

## Troubleshooting

**High error rates:**
- Check service logs for errors
- Verify database connections
- Check Kafka broker health
- Ensure sufficient resources (CPU/Memory)

**Low throughput:**
- Increase service replicas
- Adjust database connection pool size
- Optimize database indexes
- Check network latency

**Timeouts:**
- Increase timeout thresholds
- Check for database deadlocks
- Monitor connection pool exhaustion
- Verify Kafka broker performance

## Results

Test results are saved to:
- `load-tests/rest-api-results.json` - REST API test results
- `load-tests/kafka-throughput-results.json` - Kafka throughput results
- `load-tests/kafka-throughput-summary.txt` - Kafka throughput summary

Analyze results:
```bash
# View summary
cat load-tests/kafka-throughput-summary.txt

# View detailed JSON results
jq '.metrics' load-tests/rest-api-results.json
```
