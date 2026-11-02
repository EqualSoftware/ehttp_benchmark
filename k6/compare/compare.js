// Fixed-rate comparison of one route of one app (open model).
//
//   k6 run -e APP=ehttp -e ROUTE=json k6/compare/compare.js
//
// Env (defaults): BASE_URL (http://127.0.0.1:8090), APP (app), ROUTE (json),
// RATE (4000 req/s), DURATION (30s), WARMUP (10s, 0 skips it), PRE_VUS (200),
// MAX_VUS (2000), OUT (a file for the JSON summary; empty writes none).
//
// The warm-up scenario is not counted: every threshold and the summary read the
// metrics tagged scenario:measured only.
import http from 'k6/http';
import { check } from 'k6';
import { num, str, seconds } from '../lib/env.js';

const base = str('BASE_URL', 'http://127.0.0.1:8090');
const app = str('APP', 'app');
const route = str('ROUTE', 'json');
const rate = num('RATE', 4000);
const duration = str('DURATION', '30s');
const warmup = str('WARMUP', '10s');
const preVUs = num('PRE_VUS', 200);
const maxVUs = num('MAX_VUS', 2000);

const paths = {
  plaintext: { path: '/plaintext', ok: (r) => r.body === 'Hello, World!' },
  json: { path: '/json', ok: (r) => r.body === '{"message":"Hello, World!"}' },
  users: { path: '/users/42', ok: (r) => r.body === '{"id":"42"}' },
  mw: { path: '/mw/json', ok: (r) => r.body === '{"message":"Hello, World!"}' },
};
if (!paths[route]) throw new Error(`ROUTE must be one of ${Object.keys(paths).join(', ')}`);

function scenario(startTime, d) {
  return {
    executor: 'constant-arrival-rate',
    exec: 'hit',
    rate,
    timeUnit: '1s',
    duration: d,
    preAllocatedVUs: preVUs,
    maxVUs,
    startTime,
  };
}

const scenarios = {};
if (seconds(warmup) > 0) scenarios.warmup = scenario('0s', warmup);
scenarios.measured = scenario(`${seconds(warmup)}s`, duration);

// Each threshold always passes. It only makes k6 keep the measured sub-metric.
export const options = {
  discardResponseBodies: true, // hit() reads the body of 1 request in 100
  scenarios,
  thresholds: {
    'http_req_duration{scenario:measured}': ['max>=0'],
    'http_req_failed{scenario:measured}': ['rate>=0'],
    'http_reqs{scenario:measured}': ['count>=0'],
    'dropped_iterations{scenario:measured}': ['count>=0'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'p(95)', 'p(99)', 'max'],
};

const target = paths[route];
const textParams = { responseType: 'text' };
let n = 0;

export function hit() {
  // Read and check the body of 1 request in 100 only, so the generator stays light.
  if (++n % 100 === 0) {
    const res = http.get(`${base}${target.path}`, textParams);
    check(res, { 'status 200': (r) => r.status === 200, body: target.ok });
    return;
  }
  check(http.get(`${base}${target.path}`), { 'status 200': (r) => r.status === 200 });
}

function metric(data, name) {
  return (data.metrics[name] || {}).values || {};
}

export function handleSummary(data) {
  const d = metric(data, 'http_req_duration{scenario:measured}');
  const count = metric(data, 'http_reqs{scenario:measured}').count || 0;
  const result = {
    app,
    route,
    target_rate: rate,
    duration_s: seconds(duration),
    requests: count,
    rps: count / seconds(duration),
    p50_ms: d.med,
    p95_ms: d['p(95)'],
    p99_ms: d['p(99)'],
    max_ms: d.max,
    error_rate: metric(data, 'http_req_failed{scenario:measured}').rate || 0,
    dropped: metric(data, 'dropped_iterations{scenario:measured}').count || 0,
  };
  const line = `RESULT app=${app} route=${route} rps=${result.rps.toFixed(0)} ` +
    `p50=${d.med.toFixed(3)}ms p95=${d['p(95)'].toFixed(3)}ms p99=${d['p(99)'].toFixed(3)}ms ` +
    `err=${(result.error_rate * 100).toFixed(2)}% dropped=${result.dropped}\n`;
  const out = {};
  out.stdout = line;
  if (__ENV.OUT) out[__ENV.OUT] = JSON.stringify(result, null, 2);
  return out;
}
