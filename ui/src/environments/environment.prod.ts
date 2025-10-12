// Environment configuration (production)
declare global { interface Window { __kmmConfig?: { apiBaseUrl?: string }; } }
export const environment = {
  production: true,
  apiBaseUrl: (typeof window !== 'undefined' && window.__kmmConfig?.apiBaseUrl) || '/api'
};
