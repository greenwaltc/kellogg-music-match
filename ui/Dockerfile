# UI Dockerfile
# 1. Build Angular app
FROM --platform=$BUILDPLATFORM node:20-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG API_BASE_URL=""
WORKDIR /app
COPY package*.json ./
RUN npm ci --no-audit --no-fund
COPY . .
# Build production version
ENV API_BASE_URL=${API_BASE_URL}
# Inject runtime config placeholder file before build; Angular app will fetch /config.json at runtime
RUN echo "{\n  \"apiBaseUrl\": \"${API_BASE_URL}\"\n}" > src/config.json \
	&& npm run build -- --configuration=production

# 2. Nginx for static hosting
FROM nginx:1.27-alpine
# Remove default nginx static assets
RUN rm -rf /usr/share/nginx/html/*
COPY --from=build /app/dist/kellogg-music-match-ui /usr/share/nginx/html
# Copy generated config.json (can be overridden at runtime by mounting another file)
COPY --from=build /app/src/config.json /usr/share/nginx/html/config.json
# Provide a simple default nginx config enabling gzip & Angular fallback
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -q -O - http://localhost/health || exit 1
