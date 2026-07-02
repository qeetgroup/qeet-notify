import {
  Badge,
  Button,
  ScrollArea,
  Separator,
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@qeetrix/ui";
import { BellIcon } from "lucide-react";
import React, { useState } from "react";

import { NotificationItem } from "../Inbox/NotificationItem";
import type { InboxProps } from "../Inbox/Inbox";

export interface BellProps extends InboxProps {
  placement?: "right" | "left";
}

export function Bell({
  placement = "right",
  maxItems = 20,
  notifications = [],
  onNotificationClick,
}: BellProps) {
  const [open, setOpen] = useState(false);
  const unread = notifications.filter((n) => !n.read).length;

  return (
    <>
      <div className="relative inline-flex">
        <Button variant="ghost" size="icon" aria-label="Notifications" onClick={() => setOpen(true)}>
          <BellIcon className="size-5" />
        </Button>
        {unread > 0 && (
          <Badge
            variant="destructive"
            className="absolute -top-1 -right-1 flex size-5 items-center justify-center rounded-full p-0 text-[10px]"
          >
            {unread > 99 ? "99+" : unread}
          </Badge>
        )}
      </div>

      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side={placement} className="flex w-80 flex-col p-0">
          <SheetHeader className="px-4 pt-4 pb-3">
            <SheetTitle className="flex items-center gap-2">
              <BellIcon className="size-4" />
              Notifications
              {unread > 0 && (
                <Badge variant="secondary" className="ml-auto">
                  {unread} new
                </Badge>
              )}
            </SheetTitle>
          </SheetHeader>
          <Separator />
          <ScrollArea className="flex-1">
            {notifications.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-2 py-12 text-center text-muted-foreground">
                <BellIcon className="size-8 opacity-40" />
                <p className="text-sm">No notifications yet.</p>
              </div>
            ) : (
              <div className="divide-y">
                {notifications.slice(0, maxItems).map((n) => (
                  <NotificationItem
                    key={n.id}
                    notification={n}
                    onClick={(notification) => {
                      onNotificationClick?.(notification);
                      setOpen(false);
                    }}
                  />
                ))}
              </div>
            )}
          </ScrollArea>
        </SheetContent>
      </Sheet>
    </>
  );
}
