-- Products register their own kinds; the service refuses any it does not know.
ALTER TABLE notification.notifications DROP CONSTRAINT notifications_kind_check;
