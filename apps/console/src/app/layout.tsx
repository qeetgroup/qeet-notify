import type { Metadata } from "next";
import "@qeetrix/ui/qeetrix.css";
import "./globals.css";

export const metadata: Metadata = {
  title: "Qeet Notify",
  description: "Multi-channel notification management dashboard",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
