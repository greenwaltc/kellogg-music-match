import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

export interface PasswordComplexity {
  minLength: boolean;
  hasUpperCase: boolean;
  hasLowerCase: boolean;
  hasNumber: boolean;
  hasSpecialChar: boolean;
}

export function passwordComplexityValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    const value = control.value;
    
    if (!value) {
      return null; // Let required validator handle empty values
    }

    const requirements = {
      minLength: value.length >= 8,
      hasUpperCase: /[A-Z]/.test(value),
      hasLowerCase: /[a-z]/.test(value),
      hasNumber: /[0-9]/.test(value),
      hasSpecialChar: /[!@#$%^&*(),.?":{}|<>_]/.test(value)
    };

    const allMet = Object.values(requirements).every(req => req);
    
    if (allMet) {
      return null; // All requirements met
    }

    return { passwordComplexity: requirements };
  };
}

export function confirmPasswordValidator(passwordControlName: string): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control.parent) {
      return null;
    }

    const password = control.parent.get(passwordControlName);
    const confirmPassword = control;

    if (!password || !confirmPassword) {
      return null;
    }

    if (password.value !== confirmPassword.value) {
      return { passwordMismatch: true };
    }

    return null;
  };
}

export function getPasswordStrength(password: string): { score: number; label: string; color: string } {
  if (!password) {
    return { score: 0, label: 'Enter password', color: '#6b7280' };
  }

  const complexity = {
    minLength: password.length >= 8,
    hasUpperCase: /[A-Z]/.test(password),
    hasLowerCase: /[a-z]/.test(password),
    hasNumber: /[0-9]/.test(password),
    hasSpecialChar: /[!@#$%^&*(),.?":{}|<>_]/.test(password)
  };

  const score = Object.values(complexity).filter(Boolean).length;

  const strengthMap = {
    0: { label: 'Very Weak', color: '#ef4444' },
    1: { label: 'Weak', color: '#f97316' },
    2: { label: 'Fair', color: '#eab308' },
    3: { label: 'Good', color: '#84cc16' },
    4: { label: 'Strong', color: '#22c55e' },
    5: { label: 'Very Strong', color: '#16a34a' }
  };

  return { score, ...strengthMap[score as keyof typeof strengthMap] };
}