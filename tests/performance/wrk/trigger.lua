-- wrk Lua script for /v1/events throughput baseline.
-- Usage: wrk -t8 -c200 -d30s -s tests/performance/wrk/trigger.lua http://localhost:8080
-- Set WRK_API_KEY env before running (wrk reads it via os.getenv).

local counter = 0
local api_key = os.getenv("WRK_API_KEY") or "dev-api-key"

request = function()
  counter = counter + 1
  local body = string.format(
    '{"name":"wrk.load.event","subscriber_id":"wrk-sub-%d","payload":{}}',
    counter % 1000
  )
  wrk.headers["Content-Type"]    = "application/json"
  wrk.headers["X-Qeet-Api-Key"]  = api_key
  wrk.headers["Idempotency-Key"] = string.format("wrk-%d", counter)
  return wrk.format("POST", "/v1/events", nil, body)
end

done = function(summary, latency, requests)
  io.write(string.format(
    "Requests/sec: %.2f  Avg latency: %.2fms  P99: %.2fms\n",
    summary.requests / (summary.duration / 1e6),
    latency.mean / 1000,
    latency:percentile(99) / 1000
  ))
end
