-- Where a user's email notifications go; none means in-app only.
ALTER TABLE iam.users ADD COLUMN email text CHECK (char_length(email) <= 254);
