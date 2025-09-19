// Environment configuration (development)
declare global { interface Window { __kmmConfig?: { apiBaseUrl?: string }; } }

export const environment = {
  production: false,
  // Use /api proxy in Docker, direct connection in dev
  get apiBaseUrl() {
    return (typeof window !== 'undefined' && window.__kmmConfig?.apiBaseUrl) || '/api';
  }
};
