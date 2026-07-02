import {
  Avatar,
  AvatarFallback,
  DataState,
  Sheet,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TimeSince,
} from "@qeetrix/ui";
import { UsersIcon } from "lucide-react";
import { useState } from "react";

import { useSubscribers } from "../hooks/useSubscribers";
import { PreferencesPanel } from "./PreferencesPanel";

function initials(s: any): string {
  const name = [s.first_name, s.last_name].filter(Boolean).join(" ") || s.external_id;
  return name.split(" ").map((w: string) => w[0]).slice(0, 2).join("").toUpperCase();
}

export function SubscriberTable() {
  const { data, isLoading, isError, error } = useSubscribers();
  const items = (data as any)?.data ?? data ?? [];
  const [selected, setSelected] = useState<any>(null);

  return (
    <>
      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && items.length === 0}
        emptyIcon={UsersIcon}
        emptyTitle="No subscribers yet"
        emptyDescription="Subscribers are created automatically when you trigger events."
        skeletonRows={6}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Subscriber</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Phone</TableHead>
              <TableHead>Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((s: any) => (
              <TableRow key={s.id} className="cursor-pointer" onClick={() => setSelected(s)}>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Avatar className="size-7">
                      <AvatarFallback className="text-[10px]">{initials(s)}</AvatarFallback>
                    </Avatar>
                    <span className="text-sm font-medium">{s.external_id}</span>
                  </div>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">{s.email ?? "—"}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{s.phone ?? "—"}</TableCell>
                <TableCell>
                  <TimeSince value={s.created_at} className="text-sm text-muted-foreground" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>

      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        {selected && <PreferencesPanel subscriber={selected} />}
      </Sheet>
    </>
  );
}
