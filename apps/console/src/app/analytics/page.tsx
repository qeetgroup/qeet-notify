"use client";

import useSWR from "swr";

const fetcher = (url: string) => fetch(url).then((r) => r.json());

type Stat = {
  channel: string;
  event_type: string;
  count: number;
  date: string;
};

export default function AnalyticsPage() {
  const { data, isLoading } = useSWR<{ stats: Stat[] }>(
    "/api/v1/analytics/delivery",
    fetcher
  );

  if (isLoading) return <p>Loading…</p>;

  const stats = data?.stats ?? [];

  return (
    <main style={{ padding: "2rem" }}>
      <h1>Delivery Analytics (30 days)</h1>
      {stats.length === 0 ? (
        <p>No delivery data yet.</p>
      ) : (
        <table style={{ borderCollapse: "collapse", width: "100%" }}>
          <thead>
            <tr>
              {["Date", "Channel", "Event", "Count"].map((h) => (
                <th key={h} style={{ textAlign: "left", padding: "8px", borderBottom: "1px solid #eee" }}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {stats.map((s, i) => (
              <tr key={i}>
                <td style={{ padding: "8px" }}>{s.date}</td>
                <td style={{ padding: "8px" }}>{s.channel}</td>
                <td style={{ padding: "8px" }}>{s.event_type}</td>
                <td style={{ padding: "8px" }}>{s.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  );
}
