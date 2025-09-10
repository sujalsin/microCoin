import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const orderSuccessRate = new Rate('order_success_rate');
const orderErrorRate = new Rate('order_error_rate');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 }, // Ramp up
    { duration: '60s', target: 50 }, // Stay at 50 users
    { duration: '30s', target: 0 },  // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
    http_req_failed: ['rate<0.01'],   // Error rate must be below 1%
    order_success_rate: ['rate>0.99'], // Order success rate must be above 99%
  },
};

// Test data
const BASE_URL = 'http://localhost:8080';
let authToken = '';
let userId = '';

export function setup() {
  // Sign up a test user
  const signupPayload = JSON.stringify({
    email: `test-${Date.now()}@example.com`,
    password: 'testpassword123'
  });

  const signupResponse = http.post(`${BASE_URL}/auth/signup`, signupPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (signupResponse.status !== 200) {
    throw new Error(`Signup failed: ${signupResponse.status}`);
  }

  const signupData = JSON.parse(signupResponse.body);
  authToken = signupData.token;
  userId = signupData.user.id;

  // Top up the account
  const topupPayload = JSON.stringify({
    amount: '10000.00'
  });

  const topupResponse = http.post(`${BASE_URL}/api/fund/topup`, topupPayload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${authToken}`,
      'Idempotency-Key': `topup-${Date.now()}`
    },
  });

  if (topupResponse.status !== 200) {
    throw new Error(`Topup failed: ${topupResponse.status}`);
  }

  return { authToken, userId };
}

export default function(data) {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${data.authToken}`,
    'Idempotency-Key': `order-${Date.now()}-${Math.random()}`
  };

  // Create a random order
  const orderTypes = ['MARKET', 'LIMIT'];
  const orderSides = ['BUY', 'SELL'];
  const symbols = ['BTC-USD', 'ETH-USD'];
  
  const orderType = orderTypes[Math.floor(Math.random() * orderTypes.length)];
  const orderSide = orderSides[Math.floor(Math.random() * orderSides.length)];
  const symbol = symbols[Math.floor(Math.random() * symbols.length)];
  
  const orderPayload = {
    symbol: symbol,
    side: orderSide,
    type: orderType,
    qty: (Math.random() * 0.1 + 0.001).toFixed(8), // Random quantity between 0.001 and 0.101
  };

  // Add price for limit orders
  if (orderType === 'LIMIT') {
    const basePrice = symbol === 'BTC-USD' ? 60000 : 3000;
    const priceVariation = (Math.random() - 0.5) * 0.1; // ±5% variation
    orderPayload.price = (basePrice * (1 + priceVariation)).toFixed(2);
  }

  const response = http.post(`${BASE_URL}/api/orders`, JSON.stringify(orderPayload), {
    headers: headers,
  });

  // Check response
  const success = check(response, {
    'order created successfully': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
    'response has order_id': (r) => {
      if (r.status === 200) {
        const data = JSON.parse(r.body);
        return data.order_id !== undefined;
      }
      return false;
    },
  });

  // Record metrics
  orderSuccessRate.add(success);
  orderErrorRate.add(!success);

  // Log errors
  if (!success) {
    console.error(`Order failed: ${response.status} - ${response.body}`);
  }

  // Small delay between requests
  sleep(0.1);
}

export function teardown(data) {
  console.log('Load test completed');
}
