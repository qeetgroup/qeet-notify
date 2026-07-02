-- Revert to the original status set (fails if any row is 'suppressed').
ALTER TABLE notifications DROP CONSTRAINT notifications_status_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_status_check
    CHECK (status IN ('pending','queued','sent','delivered','failed','skipped'));
