import { useSignIn } from "@qeet-id/react";
import {
  Button,
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
  Input,
  Spinner,
} from "@qeetrix/ui";
import { QeetLogo } from "@qeetrix/ui/brand";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { keyStore } from "@/lib/api";
import { sessionStore } from "@/lib/auth";

export const Route = createFileRoute("/sign-in")({ component: SignInPage });

function SignInPage() {
  const navigate = useNavigate();
  const { status, signIn, verifyMfa } = useSignIn();

  const [email, setEmail]     = useState("");
  const [password, setPassword] = useState("");
  const [mfaCode, setMfaCode] = useState("");

  // Separate state for the API key step (shown after qeet-id sign-in)
  const [apiKey, setApiKey]   = useState("");
  const [apiKeyError, setApiKeyError] = useState("");
  const [step, setStep]       = useState<"identity" | "api-key">("identity");

  const isLoading = status.step === "loading";
  const authError = status.step === "error" ? status.error : "";
  const needsMfa  = status.step === "needs_mfa";

  async function handleIdentity(e: React.FormEvent) {
    e.preventDefault();
    await signIn({ email, password });
  }

  async function handleMfa(e: React.FormEvent) {
    e.preventDefault();
    await verifyMfa({ code: mfaCode });
  }

  function handleApiKey(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = apiKey.trim();
    if (!trimmed) { setApiKeyError("API key is required"); return; }
    keyStore.set(trimmed);
    navigate({ to: "/" as never, replace: true });
  }

  // Store identity on completion, then advance to API key step.
  if (status.step === "complete" && step === "identity") {
    sessionStore.set(email);
    setStep("api-key");
  }

  // --- Step 2: enter qeet-notify API key ---
  if (step === "api-key") {
    return (
      <AuthLayout>
        <div className="space-y-1 text-center">
          <h1 className="text-xl font-semibold">Connect your workspace</h1>
          <p className="text-sm text-muted-foreground">
            Enter your Qeet Notify API key to access the console.
          </p>
        </div>
        <form onSubmit={handleApiKey} className="space-y-4">
          <FieldGroup>
            <Field>
              <FieldLabel>API Key</FieldLabel>
              <Input
                type="password"
                placeholder="qn_live_…"
                value={apiKey}
                onChange={(e) => { setApiKey(e.target.value); setApiKeyError(""); }}
                autoFocus
              />
              {apiKeyError && <FieldError>{apiKeyError}</FieldError>}
            </Field>
          </FieldGroup>
          <Button type="submit" className="w-full">Continue</Button>
        </form>
        <p className="text-center text-xs text-muted-foreground">
          Find your API key in Settings → API Keys.
        </p>
      </AuthLayout>
    );
  }

  // --- Step 1: MFA code ---
  if (needsMfa) {
    return (
      <AuthLayout>
        <div className="space-y-1 text-center">
          <h1 className="text-xl font-semibold">Two-factor authentication</h1>
          <p className="text-sm text-muted-foreground">Enter your authenticator code.</p>
        </div>
        <form onSubmit={handleMfa} className="space-y-4">
          <FieldGroup>
            <Field>
              <FieldLabel>Verification code</FieldLabel>
              <Input
                type="text"
                inputMode="numeric"
                maxLength={6}
                placeholder="000000"
                value={mfaCode}
                onChange={(e) => setMfaCode(e.target.value)}
                autoFocus
              />
              {authError && <FieldError>{authError}</FieldError>}
            </Field>
          </FieldGroup>
          <Button type="submit" className="w-full" disabled={isLoading}>
            {isLoading ? <Spinner className="mr-2" /> : null}
            {isLoading ? "Verifying…" : "Verify"}
          </Button>
        </form>
      </AuthLayout>
    );
  }

  // --- Step 1: email + password ---
  return (
    <AuthLayout>
      <div className="flex flex-col items-center gap-2 text-center">
        <QeetLogo className="h-8 w-auto" />
        <h1 className="text-xl font-semibold">Sign in to Notify</h1>
        <p className="text-sm text-muted-foreground">Use your Qeet ID to continue.</p>
      </div>

      <form onSubmit={handleIdentity} className="space-y-4">
        <FieldGroup>
          <Field>
            <FieldLabel>Email</FieldLabel>
            <Input
              type="email"
              placeholder="you@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoFocus
            />
          </Field>
          <Field>
            <FieldLabel>Password</FieldLabel>
            <Input
              type="password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Field>
          {authError && <FieldError>{authError}</FieldError>}
        </FieldGroup>
        <Button type="submit" className="w-full" disabled={isLoading}>
          {isLoading ? <Spinner className="mr-2" /> : null}
          {isLoading ? "Signing in…" : "Sign in"}
        </Button>

        <FieldSeparator>or</FieldSeparator>

        <p className="text-center text-xs text-muted-foreground">
          API key only?{" "}
          <button
            type="button"
            className="underline underline-offset-2"
            onClick={() => setStep("api-key")}
          >
            Skip to API key
          </button>
        </p>
      </form>
    </AuthLayout>
  );
}

function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 px-4">
        {children}
      </div>
    </div>
  );
}
