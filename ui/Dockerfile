# UI Dockerfile
# 1. Build Angular app
FROM node:20-alpine AS build
ARG API_BASE_URL=""
WORKDIR /app

# Copy package files first for better Docker layer caching
COPY package*.json ./
RUN npm ci --no-audit --no-fund

# Copy source code
COPY . .

# Build production version
ENV API_BASE_URL=${API_BASE_URL}
# Build only; runtime config.json will be generated in final image entrypoint.
RUN echo "{\n  \"apiBaseUrl\": \"${API_BASE_URL}\"\n}" > src/config.json \
    && npm run build -- --configuration=production

# 2. Nginx for static hosting
FROM nginx:1.27-alpine
# Remove default nginx static assets
RUN rm -rf /usr/share/nginx/html/*

# Create nginx user for better security (if not exists)
RUN addgroup -g 1001 -S nginx-user && \
    adduser -S -D -H -u 1001 -h /var/cache/nginx -s /sbin/nologin -G nginx-user nginx-user

# Copy built application
COPY --from=build /app/dist/kellogg-music-match-ui /usr/share/nginx/html
# Copy generated config.json (can be overridden at runtime by mounting another file)
# Provide a temporary placeholder (will be replaced at runtime if writable)
COPY --from=build /app/src/config.json /usr/share/nginx/html/config.json
# Provide a simple default nginx config enabling gzip & Angular fallback
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set appropriate permissions
RUN chown -R nginx-user:nginx-user /usr/share/nginx/html && \
    chmod -R 755 /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -q -O - http://localhost/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
