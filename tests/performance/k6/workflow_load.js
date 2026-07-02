// k6 load test — workflow trigger with multi-step DAG.
// Run: k6 run --env API_URL=http://localhost:8080 --env API_KEY=<key> tests/performance/k6/workflow_load.js
import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 50,
  duration: "60s",
  thresholds: {
    http_req_duration: ["p(95)<1000"],
    http_req_failed: ["rate<0.01"],
  },
};

const BASE = __ENV.API_URL || "http://localhost:8080";
const KEY  = __ENV.API_KEY  || "dev-api-key";

export default function () {
  const res = http.post(
    `${BASE}/v1/events`,
    JSON.stringify({
      name: "load.workflow.event",
      subscriber_id: `wf-sub-${__VU}`,
      payload: { email: "load@example.com", phone: "+919876543210" },
    }),
    {
      headers: {
        "Content-Type": "application/json",
        "X-Qeet-Api-Key": KEY,
        "Idempotency-Key": `wf-${__VU}-${__ITER}`,
      },
    }
  );

  check(res, { "accepted": (r) => r.status === 202 });
  sleep(1);
}
