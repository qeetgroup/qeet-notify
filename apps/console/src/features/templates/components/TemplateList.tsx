import {
  Badge,
  Button,
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
import { FileTextIcon, PlusIcon } from "lucide-react";
import { useState } from "react";

import { useTemplates } from "../hooks/useTemplates";
import { TemplateForm } from "./TemplateForm";

const STATUS_VARIANT: Record<string, "default" | "outline"> = {
  published: "default",
  draft:     "outline",
};

const CHANNEL_LABELS: Record<string, string> = {
  email: "Email", sms: "SMS", whatsapp: "WhatsApp",
  inapp: "In-app", webhook: "Webhook", push: "Push",
};

export function TemplateList() {
  const { data, isLoading, isError, error } = useTemplates();
  const items = (data as any)?.data ?? data ?? [];
  const [sheetOpen, setSheetOpen] = useState(false);
  const [editing, setEditing] = useState<any>(null);

  function openCreate() { setEditing(null); setSheetOpen(true); }
  function openEdit(t: any) { setEditing(t); setSheetOpen(true); }

  return (
    <>
      <div className="flex items-center justify-between pb-4">
        <h2 className="text-sm font-medium text-muted-foreground">All templates</h2>
        <Button size="sm" onClick={openCreate}>
          <PlusIcon className="mr-1.5 size-4" /> New template
        </Button>
      </div>

      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && items.length === 0}
        emptyIcon={FileTextIcon}
        emptyTitle="No templates yet"
        emptyDescription="Create a template to start sending notifications."
        skeletonRows={5}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Channel</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Updated</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((t: any) => (
              <TableRow key={t.id} className="cursor-pointer" onClick={() => openEdit(t)}>
                <TableCell className="font-medium">{t.name}</TableCell>
                <TableCell>
                  <Badge variant="outline">{CHANNEL_LABELS[t.channel] ?? t.channel}</Badge>
                </TableCell>
                <TableCell>
                  <Badge variant={STATUS_VARIANT[t.status] ?? "outline"}>{t.status}</Badge>
                </TableCell>
                <TableCell>
                  <TimeSince value={t.updated_at ?? t.created_at} className="text-sm text-muted-foreground" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>

      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <TemplateForm template={editing} onSuccess={() => setSheetOpen(false)} />
      </Sheet>
    </>
  );
}
