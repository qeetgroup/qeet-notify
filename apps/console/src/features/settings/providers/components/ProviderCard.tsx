import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  DataState,
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@qeetrix/ui";
import { PlugIcon } from "lucide-react";
import { useState } from "react";

import { useProviders } from "../hooks/useProviders";

const CHANNEL_VARIANT: Record<string, "default" | "secondary" | "outline"> = {
  email:    "default",
  sms:      "secondary",
  whatsapp: "outline",
  push:     "outline",
};

export function ProviderCard() {
  const { data, isLoading, isError, error } = useProviders();
  const items = (data as any)?.data ?? data ?? [];
  const [editing, setEditing] = useState<any>(null);

  return (
    <>
      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && items.length === 0}
        emptyIcon={PlugIcon}
        emptyTitle="No providers configured"
        emptyDescription="Add a provider to start sending via a specific channel."
        skeletonRows={3}
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((p: any) => (
            <Card key={p.id} className="cursor-pointer transition-shadow hover:shadow-md" onClick={() => setEditing(p)}>
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-2">
                  <CardTitle className="text-sm font-medium">{p.name}</CardTitle>
                  <Badge variant={CHANNEL_VARIANT[p.channel] ?? "outline"} className="shrink-0 text-[10px]">
                    {p.channel}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-muted-foreground">{p.provider_type ?? p.type ?? "—"}</span>
                  <Badge variant={p.active ? "default" : "secondary"} className="text-[10px]">
                    {p.active ? "Active" : "Inactive"}
                  </Badge>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </DataState>

      <Sheet open={!!editing} onOpenChange={(o) => !o && setEditing(null)}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          {editing && (
            <>
              <SheetHeader>
                <SheetTitle>{editing.name}</SheetTitle>
                <SheetDescription>Provider configuration</SheetDescription>
              </SheetHeader>
              <p className="mt-4 text-sm text-muted-foreground">
                Provider editing form — wire credentials fields here.
              </p>
            </>
          )}
        </SheetContent>
      </Sheet>
    </>
  );
}
