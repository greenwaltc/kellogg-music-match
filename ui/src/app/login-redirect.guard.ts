import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

// Guard for the root/login route: if already authenticated, redirect to appropriate landing.
export const loginRedirectGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (auth.user()) {
    return router.parseUrl(auth.postAuthLanding());
  }
  return true;
};