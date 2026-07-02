import { UserButton, useUser } from "@qeet-id/react";
import { Avatar, AvatarFallback, Button } from "@qeetrix/ui";
import { LogOutIcon } from "lucide-react";

import { signOut } from "@/lib/auth";

export function HeaderUser() {
  const { isAuthenticated, user } = useUser();

  if (isAuthenticated && user) {
    return <UserButton />;
  }

  // Fallback: API-key-only session (no qeet-id user)
  const initials = "QN";
  return (
    <div className="flex items-center gap-2">
      <Avatar className="size-7">
        <AvatarFallback className="text-[10px]">{initials}</AvatarFallback>
      </Avatar>
      <Button variant="ghost" size="sm" onClick={signOut} aria-label="Sign out">
        <LogOutIcon className="size-4" />
      </Button>
    </div>
  );
}
