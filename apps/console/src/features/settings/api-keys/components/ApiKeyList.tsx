import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
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
import { KeyIcon, PlusIcon } from "lucide-react";
import { useState } from "react";

import { api } from "@/lib/api";
import { useApiKeys } from "../hooks/useApiKeys";

export function ApiKeyList() {
  const { data, isLoading, isError, error, refetch } = useApiKeys();
  const items = (data as any)?.data ?? data ?? [];

  const [createOpen, setCreateOpen] = useState(false);
  const [keyName, setKeyName]       = useState("");
  const [nameError, setNameError]   = useState("");
  const [saving, setSaving]         = useState(false);
  const [newKey, setNewKey]         = useState<string | null>(null);

  const [revokeTarget, setRevokeTarget] = useState<any>(null);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!keyName.trim()) { setNameError("Key name is required."); return; }
    setSaving(true); setNameError("");
    try {
      const res: any = await api("/v1/api-keys", { method: "POST", body: { name: keyName } });
      setNewKey(res.key ?? res.raw_key ?? null);
      setKeyName("");
      refetch();
    } catch (e: any) {
      setNameError(e?.message ?? "Failed to create key");
    } finally {
      setSaving(false);
    }
  }

  async function handleRevoke() {
    if (!revokeTarget) return;
    try {
      await api(`/v1/api-keys/${revokeTarget.id}`, { method: "DELETE" });
      refetch();
    } finally {
      setRevokeTarget(null);
    }
  }

  return (
    <>
      <div className="flex items-center justify-between pb-4">
        <h2 className="text-sm font-medium text-muted-foreground">API keys</h2>
        <Button size="sm" onClick={() => { setCreateOpen(true); setNewKey(null); }}>
          <PlusIcon className="mr-1.5 size-4" /> Create key
        </Button>
      </div>

      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && items.length === 0}
        emptyIcon={KeyIcon}
        emptyTitle="No API keys"
        emptyDescription="Create an API key to start sending notifications."
        skeletonRows={3}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Key prefix</TableHead>
              <TableHead>Created</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((k: any) => (
              <TableRow key={k.id}>
                <TableCell className="font-medium">{k.name}</TableCell>
                <TableCell className="font-mono text-sm text-muted-foreground">
                  {k.key_prefix ?? k.prefix ?? "qn_live_••••"}
                </TableCell>
                <TableCell>
                  <TimeSince value={k.created_at} className="text-sm text-muted-foreground" />
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setRevokeTarget(k)}
                  >
                    Revoke
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>

      {/* Create sheet */}
      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent side="right" className="flex w-full flex-col sm:max-w-sm">
          <SheetHeader>
            <SheetTitle>Create API key</SheetTitle>
            <SheetDescription>Give your key a descriptive name.</SheetDescription>
          </SheetHeader>

          {newKey ? (
            <div className="flex-1 space-y-4 py-4">
              <p className="text-sm text-muted-foreground">
                Copy this key now — it won't be shown again.
              </p>
              <Badge variant="secondary" className="block break-all font-mono text-xs">
                {newKey}
              </Badge>
            </div>
          ) : (
            <form onSubmit={handleCreate} className="flex flex-1 flex-col gap-4 py-4">
              <FieldGroup>
                <Field>
                  <FieldLabel>Name</FieldLabel>
                  <Input
                    value={keyName}
                    onChange={(e) => { setKeyName(e.target.value); setNameError(""); }}
                    placeholder="production"
                    autoFocus
                  />
                  {nameError && <FieldError>{nameError}</FieldError>}
                </Field>
              </FieldGroup>
            </form>
          )}

          <SheetFooter>
            {newKey ? (
              <Button onClick={() => setCreateOpen(false)}>Done</Button>
            ) : (
              <>
                <SheetClose render={<Button type="button" variant="outline" />}>Cancel</SheetClose>
                <Button type="submit" disabled={saving} onClick={handleCreate as any}>
                  {saving ? "Creating…" : "Create key"}
                </Button>
              </>
            )}
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {/* Revoke confirm */}
      <AlertDialog open={!!revokeTarget} onOpenChange={(o) => !o && setRevokeTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke API key?</AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{revokeTarget?.name}</strong> will stop working immediately.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <Button variant="destructive" onClick={handleRevoke}>Revoke</Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
