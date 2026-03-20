ALTER TABLE app_user
  ADD COLUMN IF NOT EXISTS email text,
  ADD COLUMN IF NOT EXISTS discourse_groups text;

ALTER TABLE app_user
  ALTER COLUMN username SET NOT NULL,
  ALTER COLUMN email SET NOT NULL,
  ALTER COLUMN discourse_external_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS app_user_username_uidx
  ON app_user (username);

CREATE UNIQUE INDEX IF NOT EXISTS app_user_email_uidx
  ON app_user (email);

CREATE UNIQUE INDEX IF NOT EXISTS app_user_discourse_external_id_uidx
  ON app_user (discourse_external_id);
