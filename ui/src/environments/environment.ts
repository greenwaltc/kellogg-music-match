// Environment configuration (development)
declare global { interface Window { __kmmConfig?: { apiBaseUrl?: string }; } }
export const environment = {
  production: false,
  // Default; overridden at runtime by config.json if present
  apiBaseUrl: (typeof window !== 'undefined' && window.__kmmConfig?.apiBaseUrl) || 'http://localhost:8080'
};
