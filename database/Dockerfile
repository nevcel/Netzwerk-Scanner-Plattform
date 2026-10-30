
FROM postgres:16-alpine

# Datenbanktabellen beim ersten Start automatisch erstellen.
COPY migrations/ /docker-entrypoint-initdb.d/
