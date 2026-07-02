import {
  Button,
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  Textarea,
} from "@qeetrix/ui";
import { useState } from "react";

import { api } from "@/lib/api";

const CHANNELS = ["email", "sms", "whatsapp", "inapp", "webhook"] as const;

interface TemplateFormProps {
  template?: any;
  onSuccess: () => void;
}

export function TemplateForm({ template, onSuccess }: TemplateFormProps) {
  const isEdit = !!template;
  const [name, setName]       = useState(template?.name ?? "");
  const [channel, setChannel] = useState(template?.channel ?? "email");
  const [subject, setSubject] = useState(template?.subject ?? "");
  const [body, setBody]       = useState(template?.body ?? "");
  const [error, setError]     = useState("");
  const [saving, setSaving]   = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim() || !body.trim()) { setError("Name and body are required."); return; }
    setSaving(true);
    setError("");
    try {
      if (isEdit) {
        await api(`/v1/templates/${template.id}`, { method: "PUT", body: { name, channel, subject, body } });
      } else {
        await api("/v1/templates", { method: "POST", body: { name, channel, subject, body } });
      }
      onSuccess();
    } catch (e: any) {
      setError(e?.message ?? "Save failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <SheetContent side="right" className="flex w-full flex-col sm:max-w-md">
      <SheetHeader>
        <SheetTitle>{isEdit ? "Edit template" : "New template"}</SheetTitle>
        <SheetDescription>
          {isEdit ? "Update template details and save." : "Create a notification template."}
        </SheetDescription>
      </SheetHeader>

      <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 overflow-y-auto py-4">
        <FieldGroup>
          <Field>
            <FieldLabel>Name</FieldLabel>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="welcome-email" />
          </Field>

          <Field>
            <FieldLabel>Channel</FieldLabel>
            <Select value={channel} onValueChange={setChannel}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {CHANNELS.map((c) => (
                  <SelectItem key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>

          {channel === "email" && (
            <Field>
              <FieldLabel>Subject</FieldLabel>
              <Input value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="Hello {{name}}" />
            </Field>
          )}

          <Field>
            <FieldLabel>Body</FieldLabel>
            <Textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Your message body. Use {{variable}} for dynamic content."
              rows={6}
            />
          </Field>

          {error && <FieldError>{error}</FieldError>}
        </FieldGroup>
      </form>

      <SheetFooter className="gap-2">
        <SheetClose render={<Button type="button" variant="outline" />}>Cancel</SheetClose>
        <Button type="submit" disabled={saving} onClick={handleSubmit as any}>
          {saving ? "Saving…" : isEdit ? "Save changes" : "Create template"}
        </Button>
      </SheetFooter>
    </SheetContent>
  );
}
