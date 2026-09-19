# Nginx liefert die statischen Dateien unserer Weboberfläche aus.
FROM nginx:1.27-alpine

# Eigene Nginx-Konfiguration für Frontend und API-Weiterleitung.
COPY nginx.conf /etc/nginx/conf.d/default.conf

# HTML-, CSS- und JavaScript-Dateien werden getrennt verwaltet
# und beim Image-Build in das öffentliche Webverzeichnis kopiert.
COPY html/ /usr/share/nginx/html/
COPY css/ /usr/share/nginx/html/css/
COPY js/ /usr/share/nginx/html/js/

EXPOSE 80