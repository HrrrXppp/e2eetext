# Local/dev Postgres with pg_cron (Debian-based; Alpine packages lack pg_cron).
FROM postgres:16-bookworm

RUN apt-get update \
	&& apt-get install -y --no-install-recommends postgresql-16-cron \
	&& rm -rf /var/lib/apt/lists/*
