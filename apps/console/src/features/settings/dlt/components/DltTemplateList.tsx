import {
  Button,
  DataState,
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TimeSince,
} from "@qeetrix/ui";
import { ShieldCheckIcon, PlusIcon } from "lucide-react";
import { useState } from "react";

import { api } from "@/lib/api";
import { useDlt } from "../hooks/useDlt";

export function DltTemplateList() {
  const { data, isLoading, isError, error, refetch } = useDlt();
  const items = (data as any)?.data ?? data ?? [];

  const [sheetOpen, setSheetOpen] = useState(false);
  const [peid, setPeid]           = useState("");
  const [templateId, setTemplateId] = useState("");
  const [regex, setRegex]         = useState("");
  const [formError, setFormError] = useState("");
  const [saving, setSaving]       = useState(false);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!peid.trim() || !templateId.trim()) { setFormError("PE ID and Template ID are required."); return; }
    setSaving(true); setFormError("");
    try {
      await api("/v1/dlt/templates", { method: "POST", body: { peid, template_id: templateId, regex } });
      setPeid(""); setTemplateId(""); setRegex("");
      setSheetOpen(false);
      refetch();
    } catch (e: any) {
      setFormError(e?.message ?? "Failed to create DLT template");
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <div className="flex items-center justify-between pb-4">
        <h2 className="text-sm font-medium text-muted-foreground">DLT templates (India)</h2>
        <Button size="sm" onClick={() => setSheetOpen(true)}>
          <PlusIcon className="mr-1.5 size-4" /> Add template
        </Button>
      </div>

      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && items.length === 0}
        emptyIcon={ShieldCheckIcon}
        emptyTitle="No DLT templates"
        emptyDescription="Add DLT-registered templates for TRAI-compliant SMS delivery."
        skeletonRows={3}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>PE ID</TableHead>
              <TableHead>Template ID</TableHead>
              <TableHead>Regex</TableHead>
              <TableHead>Added</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((d: any) => (
              <TableRow key={d.id}>
                <TableCell className="font-mono text-xs">{d.peid}</TableCell>
                <TableCell className="font-mono text-xs">{d.template_id}</TableCell>
                <TableCell className="max-w-xs truncate font-mono text-xs text-muted-foreground">
                  {d.regex ?? "—"}
                </TableCell>
                <TableCell>
                  <TimeSince value={d.created_at} className="text-sm text-muted-foreground" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>

      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent side="right" className="flex w-full flex-col sm:max-w-sm">
          <SheetHeader>
            <SheetTitle>Add DLT template</SheetTitle>
            <SheetDescription>Register a TRAI DLT-approved template for SMS.</SheetDescription>
          </SheetHeader>
          <form onSubmit={handleCreate} className="flex flex-1 flex-col gap-4 py-4">
            <FieldGroup>
              <Field>
                <FieldLabel>PE ID</FieldLabel>
                <Input value={peid} onChange={(e) => setPeid(e.target.value)} placeholder="1001XXXXXXXXX" />
              </Field>
              <Field>
                <FieldLabel>Template ID</FieldLabel>
                <Input value={templateId} onChange={(e) => setTemplateId(e.target.value)} placeholder="1007XXXXXXXXX" />
              </Field>
              <Field>
                <FieldLabel>Regex (optional)</FieldLabel>
                <Input value={regex} onChange={(e) => setRegex(e.target.value)} placeholder="Your OTP is \d+" />
              </Field>
              {formError && <FieldError>{formError}</FieldError>}
            </FieldGroup>
          </form>
          <SheetFooter>
            <SheetClose render={<Button type="button" variant="outline" />}>Cancel</SheetClose>
            <Button type="submit" disabled={saving} onClick={handleCreate as any}>
              {saving ? "Adding…" : "Add template"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  );
}
