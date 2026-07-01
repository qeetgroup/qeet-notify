import type { Metadata } from "next";
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@qeetrix/ui";
import { AppSidebar } from "@/components/AppSidebar";
import "@qeetrix/ui/qeetrix.css";
import "./globals.css";

export const metadata: Metadata = {
  title: "Qeet Notify",
  description: "Multi-channel notification management dashboard",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <SidebarProvider>
          <AppSidebar />
          <SidebarInset>
            <header className="flex h-12 items-center gap-2 border-b px-4">
              <SidebarTrigger className="-ml-1" />
            </header>
            <main className="flex-1 overflow-auto p-6">{children}</main>
          </SidebarInset>
        </SidebarProvider>
      </body>
    </html>
  );
}
