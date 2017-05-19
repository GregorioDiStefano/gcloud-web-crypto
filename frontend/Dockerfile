FROM nginx
RUN mkdir -p /www/data/www/data/dist/ /www/style/
COPY index.html /www/data/index.html
COPY dist/bundle.js /www/data/dist/bundle.js
COPY style/style.css /www/data/style/style.css
COPY nginx.conf /etc/nginx/nginx.conf
