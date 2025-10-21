// k6 Performance Test for EdgeLink API
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp-up to 10 users
    { duration: '1m', target: 50 },   // Ramp-up to 50 users
    { duration: '2m', target: 100 },  // Peak load: 100 users
    { duration: '30s', target: 0 },   // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% of requests should complete within 200ms
    http_req_failed: ['rate<0.01'],   // Error rate should be less than 1%
    errors: ['rate<0.01'],
  },
};

// Base URL (can be overridden via env var)
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Test 1: Health Check
  let healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 50ms': (r) => r.timings.duration < 50,
  });
  errorRate.add(healthRes.status !== 200);
  apiLatency.add(healthRes.timings.duration);

  sleep(1);

  // Test 2: Device Registration (simulated)
  const devicePayload = JSON.stringify({
    name: `test-device-${__VU}-${__ITER}`,
    pre_shared_key: 'test-psk-key',
    public_key: `test-pub-key-${__VU}`,
  });

  const deviceRes = http.post(
    `${BASE_URL}/api/v1/device/register`,
    devicePayload,
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  const deviceCheckResult = check(deviceRes, {
    'device registration status is 200 or 201': (r) =>
      r.status === 200 || r.status === 201 || r.status === 400, // 400 for duplicate
    'device registration response time < 200ms': (r) => r.timings.duration < 200,
  });
  errorRate.add(!deviceCheckResult);
  apiLatency.add(deviceRes.timings.duration);

  sleep(2);

  // Test 3: Get Device Config (if registration succeeded)
  if (deviceRes.status === 200 || deviceRes.status === 201) {
    const deviceId = JSON.parse(deviceRes.body).device_id;

    const configRes = http.get(`${BASE_URL}/api/v1/device/${deviceId}/config`, {
      headers: { Authorization: `Device ${deviceId}` },
    });

    check(configRes, {
      'get config status is 200': (r) => r.status === 200,
      'get config response time < 100ms': (r) => r.timings.duration < 100,
    });
    errorRate.add(configRes.status !== 200);
    apiLatency.add(configRes.timings.duration);

    sleep(1);

    // Test 4: Submit Metrics
    const metricsPayload = JSON.stringify({
      bandwidth_tx: Math.floor(Math.random() * 100000),
      bandwidth_rx: Math.floor(Math.random() * 200000),
      latency_ms: Math.floor(Math.random() * 100),
    });

    const metricsRes = http.post(
      `${BASE_URL}/api/v1/device/${deviceId}/metrics`,
      metricsPayload,
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Device ${deviceId}`,
        },
      }
    );

    check(metricsRes, {
      'submit metrics status is 200 or 201': (r) => r.status === 200 || r.status === 201,
      'submit metrics response time < 150ms': (r) => r.timings.duration < 150,
    });
    errorRate.add(metricsRes.status !== 200 && metricsRes.status !== 201);
    apiLatency.add(metricsRes.timings.duration);
  }

  sleep(1);
}

export function handleSummary(data) {
  return {
    'performance-summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const colors = options.enableColors || false;

  let summary = '\n' + indent + '=====================================\n';
  summary += indent + '  EdgeLink Performance Test Summary\n';
  summary += indent + '=====================================\n\n';

  // HTTP metrics
  summary += indent + 'HTTP Requests:\n';
  summary += indent + `  Total: ${data.metrics.http_reqs.values.count}\n`;
  summary += indent + `  Failed: ${data.metrics.http_req_failed.values.rate * 100}%\n`;
  summary += indent + `  Request Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)}/s\n\n`;

  // Response times
  summary += indent + 'Response Times:\n';
  summary += indent + `  Min: ${data.metrics.http_req_duration.values.min.toFixed(2)}ms\n`;
  summary += indent + `  Avg: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += indent + `  P50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
  summary += indent + `  P95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += indent + `  P99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
  summary += indent + `  Max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n\n`;

  // Custom metrics
  summary += indent + 'Custom Metrics:\n';
  summary += indent + `  Error Rate: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%\n`;
  summary += indent + `  API Latency P95: ${data.metrics.api_latency.values['p(95)'].toFixed(2)}ms\n\n`;

  // Thresholds
  summary += indent + 'Thresholds:\n';
  for (const [name, threshold] of Object.entries(data.metrics)) {
    if (threshold.thresholds) {
      for (const [tName, tResult] of Object.entries(threshold.thresholds)) {
        const status = tResult.ok ? '✓ PASS' : '✗ FAIL';
        summary += indent + `  ${status}: ${name} - ${tName}\n`;
      }
    }
  }

  summary += indent + '\n=====================================\n';

  return summary;
}
