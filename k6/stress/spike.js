// Spike test: a normal load, a sudden short peak, back to normal. It shows
// whether the app and the circuit breaker recover. Open model.
//
//   make stress-NAME APP=ehttp   (or: k6 run k6/stress/spike.js)
//
// Env (defaults): BASE_URL (http://127.0.0.1:8090), BASE_RATE (200 iterations/s),
// SPIKE_RATE (3000 iterations/s; one iteration = 4 requests), WARM (1m before
// the spike), RISE (10s, the jump), SPIKE (1m, at the peak), FALL (10s),
// RECOVER (3m, at the base rate again), PRE_VUS (50), MAX_VUS (1000).
import { num, str } from '../lib/env.js';
import { setup as mixSetup, mix } from '../lib/mix.js';

const base = num('BASE_RATE', 200);
const peak = num('SPIKE_RATE', 3000);

export const options = {
  scenarios: {
    spike: {
      executor: 'ramping-arrival-rate',
      startRate: base,
      timeUnit: '1s',
      preAllocatedVUs: num('PRE_VUS', 50),
      maxVUs: num('MAX_VUS', 1000),
      stages: [
        { duration: str('WARM', '1m'), target: base },
        { duration: str('RISE', '10s'), target: peak },
        { duration: str('SPIKE', '1m'), target: peak },
        { duration: str('FALL', '10s'), target: base },
        { duration: str('RECOVER', '3m'), target: base },
      ],
    },
  },
  thresholds: {
    checks: ['rate>0.95'],
    http_req_failed: ['rate<0.10'],
  },
};

export const setup = mixSetup;

export function spike() {
  mix();
}

export default spike;
