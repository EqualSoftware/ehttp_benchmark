// Soak test: a moderate load for a long time. It shows leaks and slow decay
// (memory, goroutines, cache growth, connections). Open model, constant rate.
//
//   make stress-NAME APP=ehttp   (or: k6 run k6/stress/soak.js)
//
// Env (defaults): BASE_URL (http://127.0.0.1:8090), RATE (500 iterations/s; one
// iteration = 7 requests), DURATION (30m; a real soak is hours), PRE_VUS (50),
// MAX_VUS (500).
import { num, str } from '../lib/env.js';
import { setup as mixSetup, mix } from '../lib/mix.js';

export const options = {
  scenarios: {
    soak: {
      executor: 'constant-arrival-rate',
      rate: num('RATE', 500),
      timeUnit: '1s',
      duration: str('DURATION', '30m'),
      preAllocatedVUs: num('PRE_VUS', 50),
      maxVUs: num('MAX_VUS', 500),
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

export const setup = mixSetup;

export function soak() {
  mix();
}

export default soak;
