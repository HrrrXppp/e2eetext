DELETE FROM oidc_providers WHERE name = 'Apple';

ALTER TABLE oidc_providers
    DROP COLUMN scopes,
    DROP COLUMN response_mode,
    DROP COLUMN client_secret_strategy;
