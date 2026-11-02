// Stress test: ramp above the average load, hold, ramp down. Closed model
// (virtual users with think time).
//
//   make stress-NAME APP=ehttp   (or: k6 run k6/stress/stress.js)
//
// Env (defaults): BASE_URL (http://127.0.0.1:8090), AVG_VUS (20), STRESS_VUS (100,
// above the average), RAMP (1m, each ramp), HOLD (3m, the plateau at STRESS_VUS),
// AVG_HOLD (1m, at AVG_VUS), THINK (0.1 s). Total = 3 x RAMP + AVG_HOLD + HOLD.
// A real stress run lasts 5 to 60 min: set RAMP and HOLD.
import { sleep } from 'k6';
import { num, str } from '../lib/env.js';
import { setup as mixSetup, mix } from '../lib/mix.js';

const avg = num('AVG_VUS', 20);
const peak = num('STRESS_VUS', 100);
const ramp = str('RAMP', '1m');
const think = num('THINK', 0.1);

export const options = {
  scenarios: {
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      gracefulRampDown: '30s',
      stages: [
        { duration: ramp, target: avg },
        { duration: str('AVG_HOLD', '1m'), target: avg },
        { duration: ramp, target: peak },
        { duration: str('HOLD', '3m'), target: peak },
        { duration: ramp, target: 0 },
      ],
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<1000'],
  },
};

export const setup = mixSetup;

export default function () {
  mix();
  sleep(think);
}
