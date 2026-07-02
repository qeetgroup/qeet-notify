export type Channel =
  | "email"
  | "sms"
  | "whatsapp"
  | "inapp"
  | "webhook"
  | "push";

export const CHANNELS: Channel[] = [
  "email",
  "sms",
  "whatsapp",
  "inapp",
  "webhook",
  "push",
];

export interface ChannelPreference {
  channel: Channel;
  enabled: boolean;
}
