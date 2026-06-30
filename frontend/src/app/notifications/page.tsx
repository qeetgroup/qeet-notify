"use client";

import useSWR from "swr";

const fetcher = (url: string) => fetch(url).then((r) => r.json());

type Notification = {
  id: string;
  channel: string;
  status: string;
  created_at: string;
};

export default function NotificationsPage() {
  const { data, isLoading } = useSWR<{ notifications: Notification[] }>(
    "/api/v1/notifications",
    fetcher
  );

  if (isLoading) return <p>Loading…</p>;
  const notifications = data?.notifications ?? [];

  return (
    <main style={{ padding: "2rem" }}>
      <h1>Notification Log</h1>
      {notifications.length === 0 ? (
        <p>No notifications yet.</p>
      ) : (
        <table style={{ borderCollapse: "collapse", width: "100%" }}>
          <thead>
            <tr>
              {["ID", "Channel", "Status", "Created"].map((h) => (
                <th key={h} style={{ textAlign: "left", padding: "8px", borderBottom: "1px solid #eee" }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {notifications.map((n) => (
              <tr key={n.id}>
                <td style={{ padding: "8px", fontFamily: "monospace", fontSize: "12px" }}>{n.id}</td>
                <td style={{ padding: "8px" }}>{n.channel}</td>
                <td style={{ padding: "8px" }}>{n.status}</td>
                <td style={{ padding: "8px" }}>{new Date(n.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </main>
  );
}
