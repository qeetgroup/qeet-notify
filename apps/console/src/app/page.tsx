import Link from "next/link";

export default function Home() {
  return (
    <main style={{ padding: "2rem" }}>
      <h1>Qeet Notify Dashboard</h1>
      <nav style={{ display: "flex", gap: "1rem", marginTop: "1rem" }}>
        <Link href="/notifications">Notifications</Link>
        <Link href="/workflows">Workflows</Link>
        <Link href="/templates">Templates</Link>
        <Link href="/subscribers">Subscribers</Link>
        <Link href="/analytics">Analytics</Link>
        <Link href="/settings/providers">Providers</Link>
        <Link href="/settings/dlt">India DLT</Link>
      </nav>
    </main>
  );
}
