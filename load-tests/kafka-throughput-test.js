import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics to track Kafka throughput
const resourcesCreated = new Counter('resources_created');
const eventsGenerated = new Counter('events_generated');
const creationDuration = new Trend('creation_duration');

// Test configuration - focus on high throughput
export const options = {
  scenarios: {
    kafka_throughput: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },   // Ramp up to 50 VUs
        { duration: '1m', target: 100 },   // Ramp up to 100 VUs
        { duration: '2m', target: 200 },   // Ramp up to 200 VUs
        { duration: '1m', target: 200 },   // Hold at 200 VUs
        { duration: '30s', target: 0 },    // Ramp down
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<1000'], // 95% of requests should be below 1s
    'http_req_failed': ['rate<0.05'],    // Failed requests should be less than 5%
    'resources_created': ['count>1000'], // Should create at least 1000 resources
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const batch = 5; // Create multiple resources per iteration

  for (let i = 0; i < batch; i++) {
    const payload = JSON.stringify({
      name: `KafkaTest-${Date.now()}-${__VU}-${i}`,
      description: `Kafka throughput test - VU:${__VU} Iter:${__ITER} Batch:${i}`,
      status: 'active',
    });

    const params = {
      headers: {
        'Content-Type': 'application/json',
      },
    };

    const startTime = new Date();
    const response = http.post(`${BASE_URL}/api/v1/resources`, payload, params);
    const duration = new Date() - startTime;

    const success = check(response, {
      'status is 201': (r) => r.status === 201,
      'response has id': (r) => JSON.parse(r.body).id !== undefined,
    });

    if (success) {
      resourcesCreated.add(1);
      // Each resource creation generates 1 Kafka event
      eventsGenerated.add(1);
      creationDuration.add(duration);
    } else {
      console.error(`Failed request: ${response.status}`);
    }

    // Small delay between batch items
    sleep(0.1);
  }

  // Update a resource (generates another Kafka event)
  if (Math.random() < 0.3) {
    const updatePayload = JSON.stringify({
      name: `Updated-${Date.now()}`,
      description: 'Updated for Kafka test',
      status: 'inactive',
    });

    // We need a resource ID, so let's create one first
    const createRes = http.post(`${BASE_URL}/api/v1/resources`, updatePayload, {
      headers: { 'Content-Type': 'application/json' },
    });

    if (createRes.status === 201) {
      const resourceId = JSON.parse(createRes.body).id;
      resourcesCreated.add(1);
      eventsGenerated.add(1);

      // Now update it
      http.put(`${BASE_URL}/api/v1/resources/${resourceId}`, updatePayload, {
        headers: { 'Content-Type': 'application/json' },
      });
      eventsGenerated.add(1);

      // Delete it (generates third event)
      http.del(`${BASE_URL}/api/v1/resources/${resourceId}`);
      eventsGenerated.add(1);
    }
  }

  sleep(0.5); // Brief pause between iterations
}

export function handleSummary(data) {
  const totalResources = data.metrics.resources_created.values.count;
  const totalEvents = data.metrics.events_generated.values.count;
  const duration = data.state.testRunDurationMs / 1000;
  const throughput = totalEvents / duration;

  console.log(`\n========================================`);
  console.log(`Kafka Throughput Test Results`);
  console.log(`========================================`);
  console.log(`Total Resources Created: ${totalResources}`);
  console.log(`Total Kafka Events Generated: ${totalEvents}`);
  console.log(`Test Duration: ${duration.toFixed(2)}s`);
  console.log(`Average Throughput: ${throughput.toFixed(2)} events/second`);
  console.log(`Average Creation Duration: ${data.metrics.creation_duration.values.avg.toFixed(2)}ms`);
  console.log(`P95 Creation Duration: ${data.metrics.creation_duration.values['p(95)'].toFixed(2)}ms`);
  console.log(`========================================\n`);

  return {
    'load-tests/kafka-throughput-results.json': JSON.stringify(data, null, 2),
    'load-tests/kafka-throughput-summary.txt': `
Kafka Throughput Test Summary
==============================

Resources Created: ${totalResources}
Events Generated: ${totalEvents}
Test Duration: ${duration.toFixed(2)}s
Throughput: ${throughput.toFixed(2)} events/sec

HTTP Metrics:
  Avg Request Duration: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms
  P95 Request Duration: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms
  P99 Request Duration: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms
  Failed Requests: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%

Throughput Analysis:
  Events per VU: ${(totalEvents / data.metrics.vus_max.values.value).toFixed(2)}
  Requests per second: ${data.metrics.http_reqs.values.rate.toFixed(2)}
  
Note: This test measures the number of Kafka events generated through
the REST API. Each resource operation (create/update/delete) generates
one event to Kafka via the Outbox pattern.
`,
  };
}
