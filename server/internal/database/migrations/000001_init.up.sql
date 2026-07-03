CREATE TABLE oidc_providers (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name    TEXT NOT NULL,
    link    TEXT NOT NULL,
    picture BYTEA
);

INSERT INTO oidc_providers (name, link)
VALUES (
    'Google',
    'https://accounts.google.com'
);

CREATE TABLE users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    oidc_provider_id UUID NOT NULL REFERENCES oidc_providers (id) ON DELETE RESTRICT,
    subject          TEXT NOT NULL,
    name             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (oidc_provider_id, subject)
);
