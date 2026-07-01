import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";

interface MutationMeta {
  silent?: boolean;
  successMessage?: string;
  successDescription?: string;
}

declare module "@tanstack/react-query" {
  interface Register {
    mutationMeta: MutationMeta;
    queryMeta: { silent?: boolean };
  }
}

function reportError(error: unknown, meta?: Record<string, unknown>) {
  if (meta?.silent) return;
  if (!(error instanceof ApiError)) return;
  if (error.status === 401 || error.status === 400 || error.status === 422) return;
  toast.error(error.message);
}

function reportSuccess(meta?: MutationMeta) {
  if (meta?.silent) return;
  toast.success(meta?.successMessage ?? "Saved", {
    description: meta?.successDescription,
  });
}

export function getContext() {
  const queryClient = new QueryClient({
    queryCache: new QueryCache({
      onError: (error, query) => reportError(error, query.meta),
    }),
    mutationCache: new MutationCache({
      onError: (error, _vars, _ctx, mutation) => reportError(error, mutation.meta),
      onSuccess: (_data, _vars, _ctx, mutation) =>
        reportSuccess(mutation.meta as MutationMeta | undefined),
    }),
  });

  return { queryClient };
}

export default function TanstackQueryProvider() {}
