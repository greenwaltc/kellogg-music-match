# Jellyfin Server - jellyfin.greenwalt.app and greenwalt.app (root domain)
server {
    # Nginx versions prior to 1.25
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    # Nginx versions 1.25+
    # listen 443 ssl;
    # listen [::]:443 ssl;
    # http2 on;

# Connections from local network
    allow 127.0.0.1;
    allow ::1;
    allow 192.168.1.0/24;

    # Cloudflare IP addrs
    allow 173.245.48.0/20;
    allow 103.21.244.0/22;
    allow 103.22.200.0/22;
    allow 103.31.4.0/22;
    allow 141.101.64.0/18;
    allow 108.162.192.0/18;
    allow 190.93.240.0/20;
    allow 188.114.96.0/20;
    allow 197.234.240.0/22;
    allow 198.41.128.0/17;
    allow 162.158.0.0/15;
    allow 104.16.0.0/13;
    allow 104.24.0.0/14;
    allow 172.64.0.0/13;
    allow 131.0.72.0/22;

    deny all;
    server_name greenwalt.app jellyfin.greenwalt.app;
    limit_rate 2m; # limits to 2MB/s or 16Mbps

    ## The default `client_max_body_size` is 1M, this might not be enough for some posters, etc.
    client_max_body_size 20M;

    # Comment next line to allow TLSv1.0 and TLSv1.1 if you have very old clients
    ssl_protocols TLSv1.3 TLSv1.2;
    ssl_certificate /etc/letsencrypt/live/greenwalt.app/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/greenwalt.app/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;
    ssl_trusted_certificate /etc/letsencrypt/live/greenwalt.app/chain.pem;
    
    # Disable SSL stapling since certificate doesn't have OCSP responder URL
    # ssl_stapling on;
    # ssl_stapling_verify on;

    # use a variable to store the upstream proxy
    set $jellyfin 127.0.0.1;

    # Security / XSS Mitigation Headers
    add_header X-Content-Type-Options "nosniff";

    # Permissions policy. May cause issues with some clients
    add_header Permissions-Policy "accelerometer=(), ambient-light-sensor=(), battery=(), bluetooth=(), camera=(), clipboard-read=(), display-capture=(), document-domain=(), encrypted-media=(), gamepad=(), geolocation=(), gyroscope=(), hid=(), idle-detection=(), interest-cohort=(), keyboard-map=(), local-fonts=(), magnetometer=(), microphone=(), payment=(), publickey-credentials-get=(), serial=(), sync-xhr=(), usb=(), xr-spatial-tracking=()" always;

    # Content Security Policy
    # See: https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
    # Enforces https content and restricts JS/CSS to origin
    # External Javascript (such as cast_sender.js for Chromecast) must be whitelisted.
    add_header Content-Security-Policy "default-src https: data: blob: ; img-src 'self' https://* ; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline' https://www.gstatic.com https://www.youtube.com blob:; worker-src 'self' blob:; connect-src 'self'; object-src 'none'; frame-ancestors 'self'; font-src 'self'";

    location / {
        # Proxy main Jellyfin traffic
        proxy_pass http://$jellyfin:8096;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Protocol $scheme;
        proxy_set_header X-Forwarded-Host $http_host;

        # Disable buffering when the nginx proxy gets very resource heavy upon streaming
        proxy_buffering off;
    }

    location /socket {
        # Proxy Jellyfin Websockets traffic
        proxy_pass http://$jellyfin:8096;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Protocol $scheme;
        proxy_set_header X-Forwarded-Host $http_host;
    }


    add_header Strict-Transport-Security "max-age=31536000" always; # managed by Certbot


    ssl_stapling on; # managed by Certbot
    ssl_stapling_verify on; # managed by Certbot

}

# Kellogg Music Match Server - kmm.greenwalt.app
server {
    # Nginx versions prior to 1.25
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    # Nginx versions 1.25+
    # listen 443 ssl;
    # listen [::]:443 ssl;
    # http2 on;

    # Connections from local network
    allow 127.0.0.1;
    allow ::1;
    allow 192.168.1.0/24;

    # Cloudflare IP addrs
    allow 173.245.48.0/20;
    allow 103.21.244.0/22;
    allow 103.22.200.0/22;
    allow 103.31.4.0/22;
    allow 141.101.64.0/18;
    allow 108.162.192.0/18;
    allow 190.93.240.0/20;
    allow 188.114.96.0/20;
    allow 197.234.240.0/22;
    allow 198.41.128.0/17;
    allow 162.158.0.0/15;
    allow 104.16.0.0/13;
    allow 104.24.0.0/14;
    allow 172.64.0.0/13;
    allow 131.0.72.0/22;

    deny all;
    server_name kmm.greenwalt.app;
    limit_rate 2m; # limits to 2MB/s or 16Mbps

    ## The default `client_max_body_size` is 1M, this might not be enough for some requests
    client_max_body_size 20M;

    # Comment next line to allow TLSv1.0 and TLSv1.1 if you have very old clients
    ssl_protocols TLSv1.3 TLSv1.2;
    ssl_certificate /etc/letsencrypt/live/greenwalt.app/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/greenwalt.app/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;
    ssl_trusted_certificate /etc/letsencrypt/live/greenwalt.app/chain.pem;
    
    # Disable SSL stapling since certificate doesn't have OCSP responder URL
    # ssl_stapling on;
    # ssl_stapling_verify on;

    # Security / XSS Mitigation Headers
    add_header X-Content-Type-Options "nosniff";
    add_header X-Frame-Options "SAMEORIGIN";

    # More permissive CSP for the music match application
    add_header Content-Security-Policy "default-src 'self' https: data: blob:; img-src 'self' https: data:; style-src 'self' 'unsafe-inline' https:; script-src 'self' 'unsafe-inline' 'unsafe-eval' https:; font-src 'self' https: data:; connect-src 'self' https: ws: wss:";

    location / {
        # Proxy Kellogg Music Match traffic to Kubernetes NodePort
        proxy_pass https://192.168.1.163:31771;
        proxy_set_header Host kmm-ui.traefik.me;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Protocol $scheme;
        proxy_set_header X-Forwarded-Host $http_host;

        # SSL settings for HTTPS backend
        proxy_ssl_verify off;
        proxy_ssl_server_name on;

        # Enable buffering for better performance with web apps
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
    }

    # WebSocket support for any real-time features
    location /ws {
        proxy_pass https://192.168.1.163:31771;
        proxy_set_header Host kmm-ui.traefik.me;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Protocol $scheme;
        proxy_set_header X-Forwarded-Host $http_host;

        # SSL settings for HTTPS backend
        proxy_ssl_verify off;
        proxy_ssl_server_name on;
    }

    add_header Strict-Transport-Security "max-age=31536000" always; # managed by Certbot

    # Disable SSL stapling since certificate doesn't have OCSP responder URL  
    # ssl_stapling on; # managed by Certbot
    # ssl_stapling_verify on; # managed by Certbot
}


server {
    if ($host ~ ^(.*\.)?greenwalt\.app$) {
        return 301 https://$host$request_uri;
    } # managed by Certbot


    listen 80;
    listen [::]:80;
# Connections from local network
    allow 192.168.1.0/24;

    # Cloudflare IP addrs
    allow 173.245.48.0/20;
    allow 103.21.244.0/22;
    allow 103.22.200.0/22;
    allow 103.31.4.0/22;
    allow 141.101.64.0/18;
    allow 108.162.192.0/18;
    allow 190.93.240.0/20;
    allow 188.114.96.0/20;
    allow 197.234.240.0/22;
    allow 198.41.128.0/17;
    allow 162.158.0.0/15;
    allow 104.16.0.0/13;
    allow 104.24.0.0/14;
    allow 172.64.0.0/13;
    allow 131.0.72.0/22;

    deny all;
 
    # HTTP to HTTPS redirect for all domains  
    server_name greenwalt.app jellyfin.greenwalt.app kmm.greenwalt.app;
    limit_rate 2m; # limits to 2MB/s or 16Mbps
    return 301 https://$host$request_uri;
}
