import { EmptyState, ScrollArea } from "@qeetrix/ui";
import { BellIcon } from "lucide-react";
import React from "react";

import { NotificationItem } from "./NotificationItem";
import type { QeetNotification } from "../../types";

export interface InboxProps {
  notifications?: QeetNotification[];
  maxItems?: number;
  onNotificationClick?: (notification: QeetNotification) => void;
}

export function Inbox({ notifications = [], maxItems = 20, onNotificationClick }: InboxProps) {
  if (notifications.length === 0) {
    return (
      <EmptyState
        icon={BellIcon}
        title="No notifications"
        description="You're all caught up. New notifications will appear here."
        className="py-12"
      />
    );
  }

  return (
    <ScrollArea className="h-[480px]">
      <div className="divide-y">
        {notifications.slice(0, maxItems).map((n) => (
          <NotificationItem key={n.id} notification={n} onClick={onNotificationClick} />
        ))}
      </div>
    </ScrollArea>
  );
}
