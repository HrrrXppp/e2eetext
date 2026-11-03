CREATE TABLE news (
    id           BIGSERIAL PRIMARY KEY,
    title        TEXT        NOT NULL,
    body         TEXT        NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_news_published ON news (published_at DESC, id DESC);
