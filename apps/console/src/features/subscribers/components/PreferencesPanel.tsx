import {
  Avatar,
  AvatarFallback,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  Separator,
  Switch,
} from "@qeetrix/ui";

const CHANNELS = ["email", "sms", "whatsapp", "inapp", "webhook", "push"] as const;
const CHANNEL_LABELS: Record<string, string> = {
  email: "Email", sms: "SMS", whatsapp: "WhatsApp",
  inapp: "In-app", webhook: "Webhook", push: "Push",
};

interface PreferencesPanelProps {
  subscriber: any;
}

export function PreferencesPanel({ subscriber: s }: PreferencesPanelProps) {
  const prefs: Record<string, boolean> = s.preferences ?? {};
  const name = [s.first_name, s.last_name].filter(Boolean).join(" ") || s.external_id;
  const initials = name.split(" ").map((w: string) => w[0]).slice(0, 2).join("").toUpperCase();

  return (
    <SheetContent side="right" className="w-full sm:max-w-sm">
      <SheetHeader className="mb-4">
        <div className="flex items-center gap-3">
          <Avatar>
            <AvatarFallback>{initials}</AvatarFallback>
          </Avatar>
          <div>
            <SheetTitle>{name}</SheetTitle>
            <SheetDescription>{s.email ?? s.phone ?? s.external_id}</SheetDescription>
          </div>
        </div>
      </SheetHeader>

      <Separator className="mb-4" />

      <h3 className="mb-3 text-sm font-medium">Channel preferences</h3>
      <div className="space-y-3">
        {CHANNELS.map((ch) => (
          <div key={ch} className="flex items-center justify-between">
            <span className="text-sm">{CHANNEL_LABELS[ch]}</span>
            <Switch
              checked={prefs[ch] !== false}
              aria-label={`${CHANNEL_LABELS[ch]} notifications`}
            />
          </div>
        ))}
      </div>
    </SheetContent>
  );
}
