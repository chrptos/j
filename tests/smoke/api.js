import http from 'k6/http';
import { check } from 'k6';

const baseURL = (__ENV.BASE_URL || 'http://127.0.0.1:18080').replace(/\/$/, '');
export const options = {
  scenarios: { smoke: { executor: 'shared-iterations', vus: 1, iterations: 1, maxDuration: '1m' } },
  thresholds: { checks: ['rate==1'] },
};

function request(method, path, status, verify = () => true, body = null) {
  const response = http.request(method, `${baseURL}${path}`, body, {
    timeout: '10s',
    headers: body === null ? {} : { 'Content-Type': 'application/json' },
    // Expected business/validation errors are successful smoke observations.
    responseCallback: http.expectedStatuses(status),
    tags: { name: `${method} ${path.split('?')[0].replace(/\/products\/\d+\//, '/products/{productId}/')}` },
  });
  let parsed;
  try { parsed = response.json(); } catch (_) { parsed = null; }
  let validBody = false;
  try { validBody = parsed !== null && verify(parsed); } catch (_) { validBody = false; }
  check(response, {
    [`${method} ${path}: status ${status}`]: (r) => r.status === status,
    [`${method} ${path}: response body`]: () => validBody,
  });
}

function problem(code, field) {
  return (body) => body.code === code && typeof body.traceId === 'string' && body.traceId.length > 0
    && (field === undefined || (Array.isArray(body.errors) && body.errors.some((e) => e.field === field)));
}

export default function () {
  request('GET', '/health/live', 200, (b) => b.status === 'ok');
  request('GET', '/health/ready', 200, (b) => b.status === 'ok');
  request('GET', '/products?keyword=Go&categoryId=1&minPrice=100&maxPrice=200&sort=price_asc&limit=1', 200,
    (b) => b.items.length === 1 && b.items[0].id === 2 && b.items[0].categoryName === 'Books'
      && b.limit === 1 && b.offset === 0 && b.hasNext === true);
  request('GET', '/products?keyword=Go&sort=price_asc&limit=1&offset=1', 200,
    (b) => b.items.length === 1 && b.items[0].id === 1 && b.offset === 1 && b.hasNext === false);
  request('GET', '/products?keyword=missing', 200, (b) => Array.isArray(b.items) && b.items.length === 0 && b.hasNext === false);
  request('GET', '/products?limit=0', 400, problem('INVALID_ARGUMENT', 'limit'));
  request('POST', '/products/1/stock/decrements', 200,
    (b) => b.productId === 1 && b.remainingStock === 9, '{"quantity":1}');
  request('POST', '/products/1/stock/decrements', 200,
    (b) => b.productId === 1 && b.remainingStock === 7, '{"quantity":2}');
  request('POST', '/products/1/stock/decrements', 400, problem('INVALID_ARGUMENT', 'quantity'), '{"quantity":0}');
  request('POST', '/products/999/stock/decrements', 404, problem('PRODUCT_NOT_FOUND'), '{"quantity":1}');
  request('POST', '/products/3/stock/decrements', 409, problem('INSUFFICIENT_STOCK'), '{"quantity":1}');
  request('POST', '/products/1/stock/decrements', 409, problem('INSUFFICIENT_STOCK'), '{"quantity":8}');
}
