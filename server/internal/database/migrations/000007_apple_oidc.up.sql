-- Provider peculiarities as data, not hardcoded Go branches: every OIDC
-- provider row now carries its own OAuth scopes, optional response_mode, and
-- client-authentication strategy, so a new provider (like Apple) is a data
-- row, not a new `if slug == "..."` branch in Go.
ALTER TABLE oidc_providers
    ADD COLUMN scopes TEXT NOT NULL DEFAULT 'openid profile',
    ADD COLUMN response_mode TEXT,
    ADD COLUMN client_secret_strategy TEXT NOT NULL DEFAULT 'static';

COMMENT ON COLUMN oidc_providers.scopes IS
    'Space-separated OAuth scopes requested for this provider (e.g. ''openid profile'').';
COMMENT ON COLUMN oidc_providers.response_mode IS
    'Optional OIDC response_mode auth param (e.g. ''form_post''); NULL to omit.';
COMMENT ON COLUMN oidc_providers.client_secret_strategy IS
    '''static'' reads a configured client_secret; ''private_key_jwt'' mints a signed client-authentication JWT per request (RFC 7523).';

-- Backfill Google's existing row explicitly, even though these values match
-- the column defaults, for clarity.
UPDATE oidc_providers
SET scopes = 'openid profile',
    client_secret_strategy = 'static'
WHERE name = 'Google';

-- Apple has no `profile` scope and requires `response_mode=form_post`
-- whenever name/email scopes are requested. It authenticates via a signed
-- JWT (private_key_jwt) instead of a static client secret.
--
-- We only request the `name` scope, not `email` — we don't keep the user's
-- email address anywhere, so there's no reason to ask Apple for it.
--
-- oidc_providers.name has no unique constraint, so this is guarded by an
-- existence check (rather than ON CONFLICT, which has no conflict target
-- here) in case an operator already seeded an Apple row by hand.
INSERT INTO oidc_providers (name, link, scopes, response_mode, client_secret_strategy)
SELECT 'Apple', 'https://appleid.apple.com', 'openid name', 'form_post', 'private_key_jwt'
WHERE NOT EXISTS (
    SELECT 1 FROM oidc_providers WHERE name = 'Apple'
);
