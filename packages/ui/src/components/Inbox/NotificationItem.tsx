import { Badge, PresenceIndicator, TimeSince, cn } from "@qeetrix/ui";
import React from "react";

import type { QeetNotification } from "../../types";

export interface NotificationItemProps {
  notification: QeetNotification;
  onClick?: (notification: QeetNotification) => void;
}

const CHANNEL_LABELS: Record<string, string> = {
  email: "Email",
  sms: "SMS",
  whatsapp: "WA",
  inapp: "In-app",
  webhook: "Hook",
  push: "Push",
};

export function NotificationItem({ notification, onClick }: NotificationItemProps) {
  return (
    <button
      type="button"
      className={cn(
        "flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        !notification.read && "bg-primary/5",
      )}
      onClick={() => onClick?.(notification)}
    >
      <div className="relative mt-0.5 shrink-0">
        {!notification.read && (
          <PresenceIndicator
            status="online"
            size="sm"
            pulse
            className="absolute -top-0.5 -right-0.5"
          />
        )}
        <div className="grid size-8 place-items-center rounded-full bg-muted text-[11px] font-semibold uppercase text-muted-foreground">
          {(CHANNEL_LABELS[notification.channel] ?? notification.channel).slice(0, 2)}
        </div>
      </div>

      <div className="min-w-0 flex-1">
        {notification.subject && (
          <p className="truncate text-sm font-medium">{notification.subject}</p>
        )}
        <p className="truncate text-sm text-muted-foreground">{notification.body}</p>
        <div className="mt-1 flex items-center gap-2">
          <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
            {CHANNEL_LABELS[notification.channel] ?? notification.channel}
          </Badge>
          <TimeSince value={notification.createdAt} className="text-xs text-muted-foreground" />
        </div>
      </div>
    </button>
  );
}
