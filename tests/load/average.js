import http from 'k6/http';
import exec from 'k6/execution';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

function integer(name, fallback, min = 1, max = 2147483647) {
  const text = __ENV[name] === undefined ? String(fallback) : __ENV[name];
  const value = Number(text);
  if (!/^\d+$/.test(text) || !Number.isSafeInteger(value) || value < min || value > max) throw new Error(`Invalid ${name}`);
  return value;
}
const config = {
  baseURL: (__ENV.BASE_URL || 'http://127.0.0.1:18082').replace(/\/$/, ''),
  products: integer('PRODUCTS', 100000, 1, 1000000),
  seed: integer('SEED', 42, 0),
  stock: integer('STOCK', 1000000),
  workload: __ENV.WORKLOAD || 'search',
  access: __ENV.ACCESS || 'uniform',
  distribution: __ENV.DISTRIBUTION || 'uniform',
  phase: __ENV.PHASE || 'average',
  rate: integer('RPS', 10),
  ramp: integer('RAMP_SECONDS', 30),
  hold: integer('HOLD_SECONDS', 300),
  warmup: integer('WARMUP_SECONDS', 120),
  vus: integer('VUS', 100),
  query: __ENV.SEARCH_QUERY || 'keyword=%E3%82%B0%E3%83%AB%E3%83%BC%E3%83%97-00&sort=id_asc&limit=20&offset=0',
};
if (!['search','decrement','mixed'].includes(config.workload) || !['uniform','hot'].includes(config.access) || !['uniform','hot'].includes(config.distribution) || !['warmup','average'].includes(config.phase)) throw new Error('Invalid workload, access or phase');
if (config.rate * (config.warmup + config.ramp + config.hold + 15) >= config.stock) throw new Error('STOCK must exceed the maximum planned decrements');
const common = { timeUnit: '1s', preAllocatedVUs: config.vus, gracefulStop: '15s' };
export const options = {
  scenarios: config.phase === 'warmup'
    ? { warmup: { ...common, executor: 'constant-arrival-rate', rate: config.rate, duration: `${config.warmup}s` } }
    : {
      ramp: { ...common, executor: 'ramping-arrival-rate', startRate: 0, stages: [{ target: config.rate, duration: `${config.ramp}s` }] },
      measure: { ...common, executor: 'constant-arrival-rate', rate: config.rate, duration: `${config.hold}s`, startTime: `${config.ramp}s` },
    },
  systemTags: ['status','method','name','scenario','expected_response','error_code'],
  thresholds: { checks: ['rate==1'], http_req_failed: ['rate==0'], dropped_iterations: ['count==0'] },
};
const requests = new Counter('load_requests');
const duration = new Trend('load_duration_ms', true);

// Iteration number determines the workload and product, independently of VU allocation.
function hash(n) {
  let x = (n + config.seed) >>> 0;
  x = Math.imul(x ^ (x >>> 16), 0x45d9f3b);
  x = Math.imul(x ^ (x >>> 16), 0x45d9f3b);
  return (x ^ (x >>> 16)) >>> 0;
}
export default function () {
  const n = exec.scenario.iterationInTest;
  const api = config.workload === 'decrement' || (config.workload === 'mixed' && n % 10 === 9) ? 'decrement' : 'search';
  const decrementIndex = config.workload === 'mixed' ? Math.floor(n / 10) : n;
  const id = config.access === 'hot'
    ? (decrementIndex % 5 !== 4 || config.products === 1 ? 1 : 2 + hash(n) % (config.products - 1))
    : 1 + hash(n) % config.products;
  const response = api === 'search'
    ? http.get(`${config.baseURL}/products?${config.query}`, { timeout: '10s', redirects: 0, tags: { name: 'GET /products' } })
    : http.post(`${config.baseURL}/products/${id}/stock/decrements`, '{"quantity":1}', { timeout: '10s', redirects: 0, headers: { 'Content-Type': 'application/json' }, tags: { name: 'POST /products/{productId}/stock/decrements' } });
  let body = null;
  try { body = response.json(); } catch (_) { /* Recorded as an invalid response below. */ }
  const valid = response.status === 200 && body !== null && (api === 'search'
    ? Array.isArray(body.items) && typeof body.hasNext === 'boolean'
    : body.productId === id && Number.isInteger(body.remainingStock) && body.remainingStock >= 0);
  check(response, { 'valid successful response': () => valid });
  const outcome = valid ? 'success' : response.status === 409 || response.status === 404 ? 'business' : 'failure';
  const known = ['INVALID_ARGUMENT','PRODUCT_NOT_FOUND','INSUFFICIENT_STOCK','DATABASE_UNAVAILABLE','DATABASE_TIMEOUT','TRANSACTION_ABORTED','INTERNAL_ERROR'];
  const code = response.status === 0 ? (response.error_code === 1050 ? 'TIMEOUT' : 'CONNECTION_ERROR') : response.status === 200 ? (valid ? 'OK' : 'INVALID_RESPONSE') : body && known.includes(body.code) ? body.code : 'UNKNOWN';
  let phase = exec.scenario.name;
  if (config.phase === 'average') {
    const start = exec.scenario.startTime + (phase === 'ramp' ? config.ramp * 1000 : 0);
    phase = Date.now() < start ? 'ramp' : Date.now() < start + config.hold * 1000 ? 'measure' : 'drain';
  }
  const tags = { api, outcome, code, phase, status: String(response.status) };
  requests.add(1, tags);
  duration.add(response.timings.duration, tags);
}
export function handleSummary() {
  return {
    [`${__ENV.RESULT_PREFIX || '/tmp/j-load'}.config.json`]: JSON.stringify(config, null, 2),
    stdout: 'Raw results: use tests/load/report.py to summarize and verify stock.\n',
  };
}
