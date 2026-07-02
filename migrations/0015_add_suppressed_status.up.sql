-- Add 'suppressed' to the notifications.status enum so the send path can mark
-- a notification that was blocked by the suppression list (Module 24).
ALTER TABLE notifications DROP CONSTRAINT notifications_status_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_status_check
    CHECK (status IN ('pending','queued','sent','delivered','failed','skipped','suppressed'));
