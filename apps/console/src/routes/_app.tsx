import {
  Separator,
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@qeetrix/ui";
import { Outlet, createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";

import { AppSidebar } from "@/components/app-sidebar";
import { ThemeToggle } from "@/components/theme-toggle";
import { NavUser } from "@/components/nav-user";
import { isAuthenticated } from "@/lib/auth";

export const Route = createFileRoute("/_app")({ component: AppLayout });

function AppLayout() {
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated()) {
      navigate({ to: "/sign-in" as never, replace: true });
    }
  }, [navigate]);

  return (
    <SidebarProvider>
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:inset-s-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-background focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:shadow-md focus:ring-2 focus:ring-ring focus:outline-none"
      >
        Skip to main content
      </a>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 border-b px-3 sm:px-4">
          <div className="flex min-w-0 items-center gap-2">
            <SidebarTrigger className="-ml-1" />
            <Separator orientation="vertical" className="mr-2 hidden h-4 lg:block" />
          </div>
          <div className="ml-auto flex shrink-0 items-center gap-1">
            <ThemeToggle />
            <Separator orientation="vertical" className="mx-1 hidden h-6 sm:block" />
            <NavUser />
          </div>
        </header>
        <main
          id="main-content"
          tabIndex={-1}
          className="flex min-w-0 flex-1 flex-col gap-4 p-4 focus:outline-none"
        >
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
