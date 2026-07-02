/** Minimal notification shape used by UI components. */
export interface QeetNotification {
  id: string;
  body: string;
  subject?: string;
  channel: string;
  status: string;
  read: boolean;
  createdAt: string;
  actionUrl?: string;
  imageUrl?: string;
}
