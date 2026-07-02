// k6 load test — POST /v1/events at target RPS.
// Run: k6 run --env API_URL=http://localhost:8080 --env API_KEY=<key> tests/performance/k6/trigger_load.js
import http from "k6/http";
import { check, sleep } from "k6";
import { Counter, Rate } from "k6/metrics";

const acceptedRate = new Rate("events_accepted");
const errorCount = new Counter("event_errors");

export const options = {
  stages: [
    { duration: "30s", target: 50 },   // ramp up to 50 VUs
    { duration: "60s", target: 200 },  // sustain 200 VUs (~1000 req/s with sleep(0.2))
    { duration: "30s", target: 0 },    // ramp down
  ],
  thresholds: {
    events_accepted: [{ threshold: "rate>0.99", abortOnFail: true }],
    http_req_duration: ["p(99)<500"],
  },
};

const BASE = __ENV.API_URL || "http://localhost:8080";
const KEY  = __ENV.API_KEY  || "dev-api-key";

export default function () {
  const res = http.post(
    `${BASE}/v1/events`,
    JSON.stringify({
      name: "load.test.event",
      subscriber_id: `sub-${Math.floor(Math.random() * 1000)}`,
      payload: { ts: Date.now() },
    }),
    {
      headers: {
        "Content-Type": "application/json",
        "X-Qeet-Api-Key": KEY,
        "Idempotency-Key": `k6-${__VU}-${__ITER}`,
      },
    }
  );

  const ok = check(res, { "status 202": (r) => r.status === 202 });
  acceptedRate.add(ok);
  if (!ok) errorCount.add(1);

  sleep(0.2);
}
