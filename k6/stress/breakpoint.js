// Breakpoint test: raise the arrival rate until a threshold fails, then abort.
// The last rate before the abort is the breakpoint. Open model: the rate does
// not slow down when the app does.
//
//   make stress-NAME APP=ehttp   (or: k6 run k6/stress/breakpoint.js)
//
// Env (defaults): BASE_URL (http://127.0.0.1:8090), START_RATE (50 iterations/s),
// MAX_RATE (8000 iterations/s; one iteration = 4 requests), DURATION (10m, the
// whole ramp), P95_MS (1000), ERROR_RATE (0.05), PRE_VUS (50), MAX_VUS (1000).
// If dropped_iterations rises, the generator is the limit: raise MAX_VUS or
// check the CPU of the host.
import { num, str } from '../lib/env.js';
import { setup as mixSetup, mix } from '../lib/mix.js';

export const options = {
  scenarios: {
    breakpoint: {
      executor: 'ramping-arrival-rate',
      startRate: num('START_RATE', 50),
      timeUnit: '1s',
      preAllocatedVUs: num('PRE_VUS', 50),
      maxVUs: num('MAX_VUS', 1000),
      stages: [{ duration: str('DURATION', '10m'), target: num('MAX_RATE', 8000) }],
    },
  },
  thresholds: {
    http_req_failed: [{ threshold: `rate<${num('ERROR_RATE', 0.05)}`, abortOnFail: true, delayAbortEval: '30s' }],
    http_req_duration: [{ threshold: `p(95)<${num('P95_MS', 1000)}`, abortOnFail: true, delayAbortEval: '30s' }],
  },
};

export const setup = mixSetup;

export function breakpoint() {
  mix();
}

export default breakpoint;
