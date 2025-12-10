import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const resourceCreationTime = new Trend('resource_creation_time');
const resourceRetrievalTime = new Trend('resource_retrieval_time');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 20 },  // Ramp up to 20 users
    { duration: '1m', target: 50 },   // Stay at 50 users
    { duration: '30s', target: 100 }, // Ramp up to 100 users
    { duration: '2m', target: 100 },  // Stay at 100 users
    { duration: '30s', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    'errors': ['rate<0.1'],              // Error rate should be less than 10%
    'http_req_failed': ['rate<0.05'],    // Failed requests should be less than 5%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Test 1: Create a resource
  const createPayload = JSON.stringify({
    name: `Resource-${Date.now()}-${Math.random()}`,
    description: `Load test resource created at ${new Date().toISOString()}`,
    status: 'active',
  });

  const createParams = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const createResponse = http.post(
    `${BASE_URL}/api/v1/resources`,
    createPayload,
    createParams
  );

  const createSuccess = check(createResponse, {
    'resource created successfully': (r) => r.status === 201,
    'create response has id': (r) => JSON.parse(r.body).id !== undefined,
  });

  errorRate.add(!createSuccess);
  resourceCreationTime.add(createResponse.timings.duration);

  if (!createSuccess) {
    console.error(`Failed to create resource: ${createResponse.status} - ${createResponse.body}`);
    sleep(1);
    return;
  }

  const resourceId = JSON.parse(createResponse.body).id;

  // Test 2: Retrieve the created resource
  const getResponse = http.get(`${BASE_URL}/api/v1/resources/${resourceId}`);

  const getSuccess = check(getResponse, {
    'resource retrieved successfully': (r) => r.status === 200,
    'retrieved resource has correct id': (r) => JSON.parse(r.body).id === resourceId,
  });

  errorRate.add(!getSuccess);
  resourceRetrievalTime.add(getResponse.timings.duration);

  // Test 3: List resources
  const listResponse = http.get(`${BASE_URL}/api/v1/resources`);

  check(listResponse, {
    'resources list retrieved': (r) => r.status === 200,
    'list contains resources': (r) => JSON.parse(r.body).length > 0,
  });

  // Test 4: Update the resource
  const updatePayload = JSON.stringify({
    name: `Updated-${resourceId}`,
    description: 'Updated during load test',
    status: 'inactive',
  });

  const updateResponse = http.put(
    `${BASE_URL}/api/v1/resources/${resourceId}`,
    updatePayload,
    createParams
  );

  check(updateResponse, {
    'resource updated successfully': (r) => r.status === 200,
  });

  // Test 5: Delete the resource (20% of the time to reduce cleanup overhead)
  if (Math.random() < 0.2) {
    const deleteResponse = http.del(`${BASE_URL}/api/v1/resources/${resourceId}`);

    check(deleteResponse, {
      'resource deleted successfully': (r) => r.status === 204,
    });
  }

  // Test 6: Health check
  const healthResponse = http.get(`${BASE_URL}/health`);

  check(healthResponse, {
    'health check passed': (r) => r.status === 200,
    'service is healthy': (r) => JSON.parse(r.body).status === 'healthy',
  });

  sleep(1); // Wait 1 second between iterations
}

export function handleSummary(data) {
  return {
    'load-tests/rest-api-results.json': JSON.stringify(data, null, 2),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;

  let summary = `\n${indent}Test Summary\n${indent}============\n\n`;

  // HTTP metrics
  summary += `${indent}HTTP Metrics:\n`;
  summary += `${indent}  Request Duration (avg): ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += `${indent}  Request Duration (p95): ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `${indent}  Request Duration (p99): ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
  summary += `${indent}  Requests per second: ${data.metrics.http_reqs.values.rate.toFixed(2)}\n`;
  summary += `${indent}  Failed requests: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%\n`;
  summary += `${indent}  Error rate: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%\n\n`;

  // Custom metrics
  summary += `${indent}Custom Metrics:\n`;
  summary += `${indent}  Resource Creation Time (avg): ${data.metrics.resource_creation_time.values.avg.toFixed(2)}ms\n`;
  summary += `${indent}  Resource Retrieval Time (avg): ${data.metrics.resource_retrieval_time.values.avg.toFixed(2)}ms\n\n`;

  // Total iterations and VUs
  summary += `${indent}Execution:\n`;
  summary += `${indent}  Iterations: ${data.metrics.iterations.values.count}\n`;
  summary += `${indent}  Virtual Users: ${data.metrics.vus.values.value}\n`;
  summary += `${indent}  Duration: ${(data.state.testRunDurationMs / 1000).toFixed(2)}s\n`;

  return summary;
}
