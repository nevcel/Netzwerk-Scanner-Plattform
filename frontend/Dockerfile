# Nginx wird ohne Root-Rechte ausgeführt.
FROM nginxinc/nginx-unprivileged:stable-alpine

# Konfiguration für die Weboberfläche und API-Weiterleitung.
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Statische Dateien der Benutzeroberfläche.
COPY html/ /usr/share/nginx/html/
COPY css/ /usr/share/nginx/html/css/
COPY js/ /usr/share/nginx/html/js/

# Interner Port des Nginx-Containers.
EXPOSE 8080