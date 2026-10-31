# Dev database image. Not for production.
# The official PostgreSQL image is Debian and already uses the PGDG apt
# repository. This adds the catalogue extensions that are not in contrib.
# Integration tests stay on stock postgres images and do not use this file.
ARG PG_MAJOR=17
FROM postgres:${PG_MAJOR}

USER root
ARG PG_MAJOR
RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
		"postgresql-${PG_MAJOR}-hypopg" \
		"postgresql-${PG_MAJOR}-pg-stat-kcache" \
		"postgresql-${PG_MAJOR}-pg-wait-sampling" \
	&& rm -rf /var/lib/apt/lists/*
# The entrypoint starts as root so it can prepare the data directory,
# then drops to the postgres user.
USER root
