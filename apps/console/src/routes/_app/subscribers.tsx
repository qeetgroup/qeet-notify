import {
  Badge,
  Button,
  EmptyState,
  Input,
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
  Skeleton,
} from "@qeetrix/ui";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PlusIcon, UsersIcon } from "lucide-react";
import { useState } from "react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/subscribers")({ component: SubscribersPage });

type Subscriber = {
  id: string;
  external_id: string;
  locale: string;
  timezone: string;
  created_at: string;
};

type SubscribersResp = { subscribers: Subscriber[]; total: number };

function SubscribersPage() {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    external_id: "",
    email: "",
    phone: "",
    locale: "en",
    timezone: "Asia/Kolkata",
  });

  const { data, isLoading } = useQuery({
    queryKey: ["subscribers"],
    queryFn: () => api<SubscribersResp>("/v1/subscribers?limit=100"),
  });

  const createMutation = useMutation({
    mutationFn: (body: typeof form) => api("/v1/subscribers", { method: "POST", body }),
    meta: { successMessage: "Subscriber added" },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["subscribers"] });
      setOpen(false);
      setForm({ external_id: "", email: "", phone: "", locale: "en", timezone: "Asia/Kolkata" });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api(`/v1/subscribers/${id}`, { method: "DELETE" }),
    meta: { successMessage: "Subscriber erased (DPDP)" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["subscribers"] }),
  });

  const subscribers = (data?.subscribers ?? []).filter(
    (s) => !search || s.external_id.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Subscribers</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {data?.total ?? 0} total subscribers
          </p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>
            <PlusIcon /> Add Subscriber
          </SheetTrigger>
          <SheetContent className="w-[480px] sm:max-w-[480px]">
            <SheetHeader>
              <SheetTitle>Add Subscriber</SheetTitle>
            </SheetHeader>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                createMutation.mutate(form);
              }}
              className="mt-6 space-y-4"
            >
              <div className="space-y-1">
                <label className="text-sm font-medium">External ID</label>
                <Input
                  value={form.external_id}
                  onChange={(e) => setForm({ ...form, external_id: e.target.value })}
                  placeholder="your-internal-user-id"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Email</label>
                <Input
                  type="email"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  placeholder="user@example.com"
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Phone (E.164)</label>
                <Input
                  value={form.phone}
                  onChange={(e) => setForm({ ...form, phone: e.target.value })}
                  placeholder="+919876543210"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Locale</label>
                  <Input
                    value={form.locale}
                    onChange={(e) => setForm({ ...form, locale: e.target.value })}
                    placeholder="en"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Timezone</label>
                  <Input
                    value={form.timezone}
                    onChange={(e) => setForm({ ...form, timezone: e.target.value })}
                    placeholder="Asia/Kolkata"
                  />
                </div>
              </div>
              <Button type="submit" disabled={createMutation.isPending} className="w-full">
                {createMutation.isPending ? "Adding…" : "Add Subscriber"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      <Input
        placeholder="Search by external ID…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 rounded-lg" />
          ))}
        </div>
      ) : subscribers.length === 0 ? (
        <EmptyState
          icon={UsersIcon}
          title="No subscribers found"
          description="Add subscribers via the API or the button above."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {subscribers.map((s) => (
            <div key={s.id} className="flex items-center justify-between px-4 py-3">
              <div>
                <p className="text-sm font-medium font-mono">{s.external_id}</p>
                <p className="text-xs text-muted-foreground">
                  {s.locale} · {s.timezone}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="outline">
                  {new Date(s.created_at).toLocaleDateString()}
                </Badge>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    if (
                      confirm(
                        "Permanently erase subscriber? This is irreversible (DPDP right to erasure).",
                      )
                    )
                      deleteMutation.mutate(s.id);
                  }}
                >
                  Erase
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
