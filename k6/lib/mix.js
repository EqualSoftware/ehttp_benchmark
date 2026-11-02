// The request mix of the stress tests: the 4 benchmark routes, the same on
// every app. Each call of mix() sends 4 requests and checks status and body.
import http from 'k6/http';
import { check } from 'k6';
import { str } from './env.js';

export const baseUrl = () => str('BASE_URL', 'http://127.0.0.1:8090');
const base = baseUrl();

const routes = [
  { path: '/plaintext', body: 'Hello, World!' },
  { path: '/json', body: '{"message":"Hello, World!"}' },
  { path: '/users/42', body: '{"id":"42"}' },
  { path: '/mw/json', body: '{"message":"Hello, World!"}' },
];

export function setup() {
  const res = http.get(`${base}/plaintext`);
  if (res.status !== 200) throw new Error(`${base} is not up (status ${res.status}). Run: make up APP=ehttp`);
}

export function mix() {
  for (const r of routes) {
    const res = http.get(`${base}${r.path}`);
    check(res, { [`${r.path} 200`]: (x) => x.status === 200, [`${r.path} body`]: (x) => x.body === r.body });
  }
}
