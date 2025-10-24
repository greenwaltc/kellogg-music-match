import { inject } from '@angular/core';
import { HttpInterceptorFn } from '@angular/common/http';

export const jwtInterceptor: HttpInterceptorFn = (req, next) => {
  // Get the token from localStorage
  const token = localStorage.getItem('affyne_token');
  
  // Don't add token to login/register requests
  const isAuthRequest = req.url.includes('/login') || req.url.includes('/register');
  
  if (token && !isAuthRequest) {
    // Clone the request and add the Authorization header
    const authReq = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
    return next(authReq);
  }
  
  // For requests without token or auth requests, proceed as normal
  return next(req);
};