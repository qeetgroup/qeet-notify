import { Button, Input } from "@qeetrix/ui";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { BellIcon } from "lucide-react";
import { useState } from "react";
import { keyStore } from "@/lib/api";

export const Route = createFileRoute("/sign-in")({ component: SignInPage });

function SignInPage() {
  const navigate = useNavigate();
  const [key, setKey] = useState("");
  const [error, setError] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = key.trim();
    if (!trimmed) {
      setError("API key is required");
      return;
    }
    keyStore.set(trimmed);
    navigate({ to: "/" as never, replace: true });
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 px-4">
        <div className="flex flex-col items-center space-y-2 text-center">
          <div className="grid size-12 place-items-center rounded-xl bg-primary text-primary-foreground">
            <BellIcon className="size-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Qeet Notify</h1>
          <p className="text-sm text-muted-foreground">
            Enter your operator API key to access the console
          </p>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="api-key" className="text-sm font-medium">
              API Key
            </label>
            <Input
              id="api-key"
              type="password"
              placeholder="qn_live_…"
              value={key}
              onChange={(e) => {
                setKey(e.target.value);
                setError("");
              }}
              autoFocus
            />
            {error && <p className="text-xs text-destructive">{error}</p>}
          </div>
          <Button type="submit" className="w-full">
            Sign in
          </Button>
        </form>
        <p className="text-center text-xs text-muted-foreground">
          API keys are managed in Settings → API Keys
        </p>
      </div>
    </div>
  );
}
