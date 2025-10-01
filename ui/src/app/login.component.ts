import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators, FormGroup, FormControl } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { AuthService } from './auth.service';
import { MatchService } from './match.service';
import { passwordComplexityValidator, confirmPasswordValidator, getPasswordStrength, PasswordComplexity } from './password-validators';

type AuthMode = 'login' | 'register' | 'forgot-password' | 'reset-password';

interface LoginFormShape { 
  username: FormControl<string | null>; 
  email: FormControl<string | null>; 
  firstName: FormControl<string | null>; 
  lastName: FormControl<string | null>;
  program: FormControl<string | null>;
  graduationYear: FormControl<number | null>;
  password: FormControl<string | null>; 
  confirmPassword: FormControl<string | null>;
  resetToken: FormControl<string | null>;
  resetEmail: FormControl<string | null>;
}

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  template: `
  <section class="auth-section">
    <div class="page-header">
      <h1>Kellogg Music Match</h1>
      <p class="tagline">Connect with classmates who share your music taste.</p>
    </div>
    
    <div class="auth-container">
      <div class="auth-toggle" *ngIf="authMode === 'login' || authMode === 'register'">
        <button 
          type="button" 
          [class.active]="!isRegisterMode" 
          (click)="toggleMode(false)">
          Login
        </button>
        <button 
          type="button" 
          [class.active]="isRegisterMode" 
          (click)="toggleMode(true)">
          Register
        </button>
      </div>

      <div class="form-card" *ngIf="authMode === 'login' || authMode === 'register'">
        <div class="form-header">
          <h2>{{ isRegisterMode ? 'Create Account' : 'Sign In' }}</h2>
        </div>
        
        <form class="auth-form" [formGroup]="form" (ngSubmit)="submit()" novalidate>
          <div class="field">
            <label for="username">Username</label>
            <input id="username" type="text" formControlName="username" placeholder="Username" />
            <div class="error" *ngIf="form.get('username')?.touched && form.get('username')?.invalid">Username required (min 3 chars)</div>
          </div>

          <div class="field" *ngIf="isRegisterMode">
            <label for="email">Email</label>
            <input id="email" type="email" formControlName="email" placeholder="Kellogg Student Email" />
            <div class="error" *ngIf="form.get('email')?.touched && form.get('email')?.invalid">Valid email required</div>
          </div>

          <div class="name-fields" *ngIf="isRegisterMode">
            <div class="field">
              <label for="firstName">First Name</label>
              <input id="firstName" type="text" formControlName="firstName" placeholder="First Name" />
              <div class="error" *ngIf="form.get('firstName')?.touched && form.get('firstName')?.invalid">First name required</div>
            </div>
            
            <div class="field">
              <label for="lastName">Last Name</label>
              <input id="lastName" type="text" formControlName="lastName" placeholder="Last Name" />
              <div class="error" *ngIf="form.get('lastName')?.touched && form.get('lastName')?.invalid">Last name required</div>
            </div>
          </div>

          <div class="program-fields" *ngIf="isRegisterMode">
            <div class="field">
              <label for="program">Program</label>
              <select id="program" formControlName="program">
                <option value="">Select your program</option>
                <option value="2Y">2Y</option>
                <option value="1Y">1Y</option>
                <option value="MMM">MMM</option>
                <option value="MBAi">MBAi</option>
                <option value="JD-MBA">JD-MBA</option>
                <option value="MD-MBA">MD-MBA</option>
                <option value="EWMBA">EWMBA</option>
                <option value="JV">JV</option>
              </select>
              <div class="error" *ngIf="form.get('program')?.touched && form.get('program')?.invalid">Program selection required</div>
            </div>
            
            <div class="field">
              <label for="graduationYear">Graduation Year</label>
              <input id="graduationYear" type="number" formControlName="graduationYear" placeholder="2027" min="2025" max="2030" />
              <div class="error" *ngIf="form.get('graduationYear')?.touched && form.get('graduationYear')?.invalid">
                Valid graduation year required (2025-2030)
              </div>
            </div>
          </div>

          <div class="field">
            <label for="password">Password</label>
            <div class="password-input-container">
              <input id="password" [type]="showPassword ? 'text' : 'password'" formControlName="password" placeholder="Password" (input)="onPasswordInput()" />
              <button type="button" class="password-toggle" (click)="togglePasswordVisibility()">
                {{ showPassword ? 'Hide' : 'Show' }}
              </button>
            </div>
            
            <!-- Password requirements (only show during registration) -->
            <div class="password-requirements" *ngIf="isRegisterMode">
              <div class="requirement-header">Password Requirements:</div>
              <div class="requirements-list">
                <div class="requirement" [class.met]="passwordComplexity.minLength">
                  <span class="requirement-icon">{{ passwordComplexity.minLength ? '✓' : '○' }}</span>
                  At least 8 characters
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasUpperCase">
                  <span class="requirement-icon">{{ passwordComplexity.hasUpperCase ? '✓' : '○' }}</span>
                  One uppercase letter
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasLowerCase">
                  <span class="requirement-icon">{{ passwordComplexity.hasLowerCase ? '✓' : '○' }}</span>
                  One lowercase letter
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasNumber">
                  <span class="requirement-icon">{{ passwordComplexity.hasNumber ? '✓' : '○' }}</span>
                  One number
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasSpecialChar">
                  <span class="requirement-icon">{{ passwordComplexity.hasSpecialChar ? '✓' : '○' }}</span>
                  One special character (!&#64;#$%^&*_)
                </div>
              </div>
              
              <!-- Password strength indicator -->
              <div class="password-strength" *ngIf="passwordStrength.score > 0">
                <div class="strength-bar">
                  <div class="strength-fill" 
                       [style.width.%]="(passwordStrength.score / 5) * 100"
                       [style.background-color]="passwordStrength.color"></div>
                </div>
                <span class="strength-label" [style.color]="passwordStrength.color">
                  {{ passwordStrength.label }}
                </span>
              </div>
            </div>
            
            <div class="error" *ngIf="form.get('password')?.touched && form.get('password')?.invalid && !isRegisterMode">
              Password required (min 6 chars)
            </div>
          </div>

          <div class="field" *ngIf="isRegisterMode">
            <label for="confirmPassword">Confirm Password</label>
            <div class="password-input-container">
              <input id="confirmPassword" [type]="showConfirmPassword ? 'text' : 'password'" formControlName="confirmPassword" placeholder="Confirm Password" (input)="onConfirmPasswordInput()" />
              <button type="button" class="password-toggle" (click)="toggleConfirmPasswordVisibility()">
                {{ showConfirmPassword ? 'Hide' : 'Show' }}
              </button>
            </div>
            <div class="error" *ngIf="showPasswordMismatchError()">
              Passwords do not match
            </div>
          </div>

          <button type="submit" class="primary-btn" [disabled]="auth.loading() || (isRegisterMode && !isPasswordValid())">
            {{ auth.loading() ? 'Processing...' : (isRegisterMode ? 'Create Account' : 'Sign In') }}
          </button>
          
          <div class="forgot-password-link" *ngIf="!isRegisterMode">
            <button type="button" class="link-btn" (click)="showForgotPassword()">
              Forgot your password?
            </button>
          </div>
          
          <div class="error form-error" *ngIf="auth.error()">{{ auth.error() }}</div>
        </form>
      </div>

      <!-- Forgot Password Form -->
      <div class="form-card" *ngIf="authMode === 'forgot-password'">
        <div class="form-header">
          <h2>Reset Password</h2>
          <p>Enter your email address and we'll send you a link to reset your password.</p>
        </div>
        
        <form class="auth-form" [formGroup]="form" (ngSubmit)="submitForgotPassword()" novalidate>
          <div class="field">
            <label for="resetEmail">Email Address</label>
            <input id="resetEmail" type="email" formControlName="resetEmail" placeholder="Enter your email" />
            <div class="error" *ngIf="form.get('resetEmail')?.touched && form.get('resetEmail')?.invalid">Valid email required</div>
          </div>

          <button type="submit" class="primary-btn" [disabled]="auth.loading() || !isForgotPasswordFormValid()">
            {{ auth.loading() ? 'Sending...' : 'Send Reset Link' }}
          </button>
          
          <div class="success-message" *ngIf="resetSuccessMessage">{{ resetSuccessMessage }}</div>
          <div class="error form-error" *ngIf="auth.error()">{{ auth.error() }}</div>
          
          <div class="back-to-login">
            <button type="button" class="link-btn" (click)="showLogin()">
              ← Back to Login
            </button>
          </div>
        </form>
      </div>

      <!-- Reset Password Form -->
      <div class="form-card" *ngIf="authMode === 'reset-password'">
        <div class="form-header">
          <h2>Set New Password</h2>
          <p>Enter your reset token and new password.</p>
        </div>
        
        <form class="auth-form" [formGroup]="form" (ngSubmit)="submitResetPassword()" novalidate>
          <div class="field">
            <label for="resetToken">Reset Token</label>
            <input id="resetToken" type="text" formControlName="resetToken" placeholder="Enter reset token from email" />
            <div class="error" *ngIf="form.get('resetToken')?.touched && !form.get('resetToken')?.value">Reset token required</div>
          </div>

          <div class="field">
            <label for="resetPassword">New Password</label>
            <div class="password-input-container">
              <input id="resetPassword" [type]="showPassword ? 'text' : 'password'" formControlName="password" placeholder="Enter new password" (input)="onPasswordInput()" />
              <button type="button" class="password-toggle" (click)="togglePasswordVisibility()">
                {{ showPassword ? 'Hide' : 'Show' }}
              </button>
            </div>
            
            <!-- Password requirements -->
            <div class="password-requirements">
              <div class="requirement-header">Password Requirements:</div>
              <div class="requirements-list">
                <div class="requirement" [class.met]="passwordComplexity.minLength">
                  <span class="requirement-icon">{{ passwordComplexity.minLength ? '✓' : '○' }}</span>
                  At least 8 characters
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasUpperCase">
                  <span class="requirement-icon">{{ passwordComplexity.hasUpperCase ? '✓' : '○' }}</span>
                  One uppercase letter
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasLowerCase">
                  <span class="requirement-icon">{{ passwordComplexity.hasLowerCase ? '✓' : '○' }}</span>
                  One lowercase letter
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasNumber">
                  <span class="requirement-icon">{{ passwordComplexity.hasNumber ? '✓' : '○' }}</span>
                  One number
                </div>
                <div class="requirement" [class.met]="passwordComplexity.hasSpecialChar">
                  <span class="requirement-icon">{{ passwordComplexity.hasSpecialChar ? '✓' : '○' }}</span>
                  One special character (!&#64;#$%^&*_)
                </div>
              </div>
            </div>
          </div>

          <button type="submit" class="primary-btn" [disabled]="auth.loading() || !form.get('resetToken')?.value || !form.get('password')?.value || !isResetPasswordValid()">
            {{ auth.loading() ? 'Resetting...' : 'Reset Password' }}
          </button>
          
          <div class="success-message" *ngIf="resetSuccessMessage">{{ resetSuccessMessage }}</div>
          <div class="error form-error" *ngIf="auth.error()">{{ auth.error() }}</div>
          
          <div class="back-to-login">
            <button type="button" class="link-btn" (click)="showLogin()">
              ← Back to Login
            </button>
          </div>
        </form>
      </div>
      
      <p class="legal">By continuing you agree to participate in music matching.</p>
    </div>
  </section>
  `,
  styles: [
    `:host { 
      display: block; 
    }
    
    .auth-section {
      max-width: 600px;
      margin: 0 auto;
      padding: 2rem 1rem;
    }

    .page-header {
      text-align: center;
      margin-bottom: 2rem;
    }

    .page-header h1 {
      margin: 0 0 0.5rem 0;
      font-size: 2.5rem;
      font-weight: 700;
      background: var(--color-gradient);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }

    .tagline {
      margin: 0;
      color: var(--color-text-muted);
      font-size: 1.1rem;
      line-height: 1.6;
    }

    .auth-container {
      max-width: 480px;
      margin: 0 auto;
    }
  
    .auth-toggle { 
      display: flex; 
      margin: 0 auto 2rem; 
      width: 100%; 
      background: var(--color-surface); 
      border: 1px solid var(--color-border); 
      border-radius: 12px; 
      padding: 4px; 
    }
    
    .auth-toggle button { 
      flex: 1; 
      padding: 0.75rem 1rem; 
      background: transparent; 
      border: none; 
      color: var(--color-text-muted); 
      font-size: 0.9rem; 
      font-weight: 500; 
      border-radius: 8px; 
      cursor: pointer; 
      transition: all 0.2s ease; 
    }
    
    .auth-toggle button.active { 
      background: var(--color-accent); 
      color: white; 
    }
    
    .auth-toggle button:hover:not(.active) { 
      color: var(--color-text); 
    }

    .form-card {
      background: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 12px;
      padding: 2rem;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      margin-bottom: 1.5rem;
    }

    .form-header {
      margin-bottom: 1.5rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid var(--color-border);
    }

    .form-header h2 {
      margin: 0;
      font-size: 1.25rem;
      font-weight: 600;
      color: var(--color-text);
    }
    
    .auth-form {
      display: flex;
      flex-direction: column;
    }
    
    .field { 
      margin-bottom: 1.5rem; 
    }
    
    .field label {
      display: block;
      margin-bottom: 0.5rem;
      font-weight: 500;
      color: var(--color-text);
      font-size: 0.9rem;
    }
    
    .field input { 
      width: 100%; 
      padding: 0.75rem;
      border: 1px solid var(--color-border);
      border-radius: 8px;
      background-color: var(--color-input-bg);
      color: var(--color-text);
      font-size: 1rem;
      transition: border-color 0.2s ease, box-shadow 0.2s ease;
    }

    .field input:focus {
      outline: none;
      border-color: var(--color-accent);
      box-shadow: 0 0 0 3px rgba(106, 42, 185, 0.1);
    }

    .field input::placeholder {
      color: var(--color-text-muted);
    }
    
    /* Name fields side by side */
    .name-fields { 
      display: grid; 
      grid-template-columns: 1fr 1fr; 
      gap: 1rem; 
      margin-bottom: 1.5rem; 
    }
    .name-fields .field { 
      margin-bottom: 0; 
    }

    /* Program fields */
    .program-fields { 
      display: grid; 
      grid-template-columns: 1fr 1fr; 
      gap: 1rem; 
      margin-bottom: 1.5rem; 
    }
    .program-fields .field { 
      margin-bottom: 0; 
    }
    
    select {
      width: 100%;
      padding: 0.75rem 1rem;
      font-size: 1rem;
      border: 2px solid #e5e7eb;
      border-radius: 0.5rem;
      background-color: var(--color-surface);
      color: var(--color-text);
    }
    
    select:focus {
      outline: none;
      border-color: #4338ca;
    }
    
    /* Password input with toggle */
    .password-input-container { 
      position: relative; 
    }
    .password-input-container input { 
      padding-right: 4rem; 
    }
    .password-toggle { 
      position: absolute; 
      right: 0.75rem; 
      top: 50%; 
      transform: translateY(-50%); 
      background: none; 
      border: none; 
      cursor: pointer; 
      font-size: 0.8rem; 
      color: var(--color-text-muted); 
      transition: color 0.2s ease; 
      font-weight: 500; 
    }
    .password-toggle:hover { 
      color: var(--color-text); 
    }
    .password-toggle:focus { 
      outline: none; 
      color: var(--color-accent); 
    }
    
    /* Password requirements styling */
    .password-requirements { 
      margin-top: 0.75rem; 
      padding: 1rem; 
      background: var(--color-bg); 
      border: 1px solid var(--color-border); 
      border-radius: 8px; 
      font-size: 0.85rem; 
    }
    .requirement-header { 
      font-weight: 600; 
      color: var(--color-text); 
      margin-bottom: 0.75rem; 
    }
    .requirements-list { 
      display: flex; 
      flex-direction: column; 
      gap: 0.5rem; 
    }
    .requirement { 
      display: flex; 
      align-items: center; 
      gap: 0.5rem; 
      color: var(--color-text-muted); 
      transition: color 0.2s ease; 
    }
    .requirement.met { 
      color: var(--color-accent); 
    }
    .requirement-icon { 
      font-weight: bold; 
      width: 1rem; 
      text-align: center; 
    }
    
    /* Password strength indicator */
    .password-strength { 
      margin-top: 0.75rem; 
    }
    .strength-bar { 
      height: 4px; 
      background: var(--color-border); 
      border-radius: 2px; 
      overflow: hidden; 
      margin-bottom: 0.5rem; 
    }
    .strength-fill { 
      height: 100%; 
      transition: width 0.3s ease, background-color 0.3s ease; 
      border-radius: 2px; 
    }
    .strength-label { 
      font-size: 0.8rem; 
      font-weight: 600; 
    }
    
    .primary-btn { 
      width: 100%; 
      padding: 0.75rem 1.5rem;
      background: var(--color-gradient);
      color: white;
      border: none;
      border-radius: 8px;
      font-size: 1rem;
      font-weight: 600;
      cursor: pointer;
      transition: transform 0.2s ease, box-shadow 0.2s ease;
      margin-top: 1rem;
    }
    
    .primary-btn:hover:not([disabled]) { 
      transform: translateY(-2px); 
      box-shadow: 0 4px 16px rgba(106, 42, 185, 0.3);
    }
    
    .primary-btn:active { 
      transform: translateY(0); 
    }
    
    .primary-btn:disabled { 
      opacity: 0.6; 
      cursor: not-allowed; 
      transform: none; 
    }

    .error { 
      color: var(--color-error); 
      font-size: 0.85rem; 
      margin-top: 0.25rem; 
    }

    .success-message {
      color: var(--color-accent);
      font-size: 0.9rem;
      margin-top: 0.5rem;
      padding: 0.75rem;
      background: var(--color-accent-light, #e0f2fe);
      border-radius: 6px;
      border: 1px solid var(--color-accent);
    }

    .back-to-login {
      text-align: center;
      margin-top: 1rem;
    }

    .link-btn {
      background: none;
      border: none;
      color: var(--color-accent);
      cursor: pointer;
      font-size: 0.9rem;
      text-decoration: underline;
      padding: 0;
    }

    .link-btn:hover {
      color: var(--color-accent-dark, #1976d2);
    }

    .forgot-password-link {
      text-align: center;
      margin-top: 1rem;
    }    .form-error { 
      margin-top: 1rem; 
      text-align: center; 
    }

    .legal { 
      text-align: center;
      margin-top: 1.5rem; 
      font-size: 0.8rem; 
      color: var(--color-text-muted); 
      letter-spacing: 0.02em; 
    }
    
    @media (max-width: 600px) { 
      .auth-section {
        padding: 1rem;
      }
      
      .page-header h1 { 
        font-size: 2rem; 
      } 
      
      .form-card { 
        padding: 1.5rem; 
      } 
      
      .name-fields, .program-fields { 
        grid-template-columns: 1fr; 
        gap: 0; 
      }
      
      .name-fields .field, .program-fields .field { 
        margin-bottom: 1.5rem; 
      }
    }
    `
  ]
})
export class LoginComponent {
  form!: FormGroup<LoginFormShape>;
  isRegisterMode = false;
  authMode: AuthMode = 'login';
  showPassword = false;
  showConfirmPassword = false;
  passwordComplexity: PasswordComplexity = {
    minLength: false,
    hasUpperCase: false,
    hasLowerCase: false,
    hasNumber: false,
    hasSpecialChar: false
  };
  passwordStrength = { score: 0, label: 'Enter password', color: '#6b7280' };
  resetSuccessMessage = '';

  constructor(private fb: FormBuilder, public auth: AuthService, private router: Router, private matches: MatchService, private route: ActivatedRoute) {
    this.initializeForm();
    
    // Check for reset token in query parameters
    this.route.queryParams.subscribe(params => {
      if (params['token']) {
        this.authMode = 'reset-password';
        this.form.get('resetToken')?.setValue(params['token']);
        this.updateValidators();
      }
    });
  }

  private initializeForm(): void {
    this.form = this.fb.group<LoginFormShape>({
      username: this.fb.control<string | null>('', [Validators.required, Validators.minLength(3)]),
      email: this.fb.control<string | null>('', [Validators.required, Validators.email]),
      firstName: this.fb.control<string | null>('', [Validators.required, Validators.minLength(2)]),
      lastName: this.fb.control<string | null>('', [Validators.required, Validators.minLength(2)]),
      program: this.fb.control<string | null>('', [Validators.required]),
      graduationYear: this.fb.control<number | null>(null, [Validators.required, Validators.min(2025), Validators.max(2030)]),
      password: this.fb.control<string | null>('', [Validators.required, Validators.minLength(6)]),
      confirmPassword: this.fb.control<string | null>('', [Validators.required]),
      resetToken: this.fb.control<string | null>('', []),
      resetEmail: this.fb.control<string | null>('', [])
    });
    this.updateValidators();
  }

  toggleMode(isRegister: boolean): void {
    this.isRegisterMode = isRegister;
    this.authMode = isRegister ? 'register' : 'login';
    this.auth.error.set(null);
    this.form.reset();
    this.passwordComplexity = {
      minLength: false,
      hasUpperCase: false,
      hasLowerCase: false,
      hasNumber: false,
      hasSpecialChar: false
    };
    this.passwordStrength = { score: 0, label: 'Enter password', color: '#6b7280' };
    this.updateValidators();
  }

  private updateValidators(): void {
    const emailControl = this.form.get('email');
    const firstNameControl = this.form.get('firstName');
    const lastNameControl = this.form.get('lastName');
    const programControl = this.form.get('program');
    const graduationYearControl = this.form.get('graduationYear');
    const passwordControl = this.form.get('password');
    const confirmPasswordControl = this.form.get('confirmPassword');
    const resetEmailControl = this.form.get('resetEmail');
    const resetTokenControl = this.form.get('resetToken');
    
    // Clear all validators first
    emailControl?.clearValidators();
    firstNameControl?.clearValidators();
    lastNameControl?.clearValidators();
    programControl?.clearValidators();
    graduationYearControl?.clearValidators();
    passwordControl?.clearValidators();
    confirmPasswordControl?.clearValidators();
    resetEmailControl?.clearValidators();
    resetTokenControl?.clearValidators();
    
    if (this.authMode === 'register' || this.isRegisterMode) {
      emailControl?.setValidators([Validators.required, Validators.email]);
      firstNameControl?.setValidators([Validators.required, Validators.minLength(2)]);
      lastNameControl?.setValidators([Validators.required, Validators.minLength(2)]);
      programControl?.setValidators([Validators.required]);
      graduationYearControl?.setValidators([Validators.required, Validators.min(2025), Validators.max(2030)]);
      passwordControl?.setValidators([Validators.required, passwordComplexityValidator()]);
      confirmPasswordControl?.setValidators([Validators.required, confirmPasswordValidator('password')]);
    } else if (this.authMode === 'login') {
      passwordControl?.setValidators([Validators.required, Validators.minLength(6)]);
    } else if (this.authMode === 'forgot-password') {
      resetEmailControl?.setValidators([Validators.required, Validators.email]);
    } else if (this.authMode === 'reset-password') {
      resetTokenControl?.setValidators([Validators.required]);
      passwordControl?.setValidators([Validators.required, passwordComplexityValidator()]);
    }
    
    // Update validity for all controls
    emailControl?.updateValueAndValidity();
    firstNameControl?.updateValueAndValidity();
    lastNameControl?.updateValueAndValidity();
    programControl?.updateValueAndValidity();
    graduationYearControl?.updateValueAndValidity();
    passwordControl?.updateValueAndValidity();
    confirmPasswordControl?.updateValueAndValidity();
    resetEmailControl?.updateValueAndValidity();
    resetTokenControl?.updateValueAndValidity();
  }

  onPasswordInput(): void {
    const password = this.form.get('password')?.value || '';
    
    this.passwordComplexity = {
      minLength: password.length >= 8,
      hasUpperCase: /[A-Z]/.test(password),
      hasLowerCase: /[a-z]/.test(password),
      hasNumber: /[0-9]/.test(password),
      hasSpecialChar: /[!@#$%^&*(),.?":{}|<>_]/.test(password)
    };

    this.passwordStrength = getPasswordStrength(password);

    // Trigger validation for confirm password if it has a value
    const confirmPasswordControl = this.form.get('confirmPassword');
    if (confirmPasswordControl?.value) {
      confirmPasswordControl.updateValueAndValidity();
    }
  }

  isPasswordValid(): boolean {
    if (!this.isRegisterMode) return true;
    
    const passwordControl = this.form.get('password');
    const confirmPasswordControl = this.form.get('confirmPassword');
    
    return !!(
      passwordControl?.valid && 
      confirmPasswordControl?.valid &&
      Object.values(this.passwordComplexity).every(req => req)
    );
  }

  submit(): void {
    if (this.form.invalid) { 
      this.form.markAllAsTouched(); 
      return; 
    }
    
    const { username, email, firstName, lastName, program, graduationYear, password } = this.form.value;
    
    if (this.isRegisterMode) {
      const registerData = {
        username: username || '',
        email: email || '',
        firstName: firstName || '',
        lastName: lastName || '',
        program: program || '',
        graduationYear: graduationYear || 0,
        password: password || ''
      };
      
      this.auth.register(registerData).subscribe({
        next: () => {
          this.matches.fetchIfReady();
          const hasArtists = !!this.auth.user()?.artists?.length;
          this.router.navigateByUrl(hasArtists ? '/matches' : '/artists');
        }
      });
    } else {
      const loginData = {
        username: username || '',
        password: password || ''
      };
      
      this.auth.login(loginData).subscribe({
        next: () => {
          this.matches.fetchIfReady();
          const hasArtists = !!this.auth.user()?.artists?.length;
          this.router.navigateByUrl(hasArtists ? '/matches' : '/artists');
        }
      });
    }
  }

  onConfirmPasswordInput(): void {
    // Trigger validation for confirm password
    const confirmPasswordControl = this.form.get('confirmPassword');
    if (confirmPasswordControl) {
      confirmPasswordControl.updateValueAndValidity();
    }
  }

  showPasswordMismatchError(): boolean {
    const password = this.form.get('password')?.value;
    const confirmPassword = this.form.get('confirmPassword')?.value;
    
    // Show error if confirm password has content and doesn't match password
    return !!(confirmPassword && password !== confirmPassword);
  }

  togglePasswordVisibility(): void {
    this.showPassword = !this.showPassword;
  }

  toggleConfirmPasswordVisibility(): void {
    this.showConfirmPassword = !this.showConfirmPassword;
  }

  showForgotPassword(): void {
    this.authMode = 'forgot-password';
    this.auth.error.set(null);
    this.resetSuccessMessage = '';
    this.updateValidators();
    console.log('Switched to forgot password mode, resetEmail validators:', this.form.get('resetEmail')?.validator);
  }

  showLogin(): void {
    this.authMode = 'login';
    this.isRegisterMode = false;
    this.auth.error.set(null);
    this.resetSuccessMessage = '';
    this.updateValidators();
  }

  submitForgotPassword(): void {
    const email = this.form.get('resetEmail')?.value;
    if (!email) return;

    this.auth.forgotPassword(email).subscribe({
      next: (response) => {
        this.resetSuccessMessage = response.message;
      },
      error: () => {
        // Error handling is done in the service
      }
    });
  }

  submitResetPassword(): void {
    const token = this.form.get('resetToken')?.value;
    const password = this.form.get('password')?.value;
    if (!token || !password) return;

    this.auth.resetPassword(token, password).subscribe({
      next: (response) => {
        this.resetSuccessMessage = response.message;
        setTimeout(() => {
          this.showLogin();
        }, 2000);
      },
      error: () => {
        // Error handling is done in the service
      }
    });
  }

  isResetPasswordValid(): boolean {
    return Object.values(this.passwordComplexity).every(req => req);
  }

  isForgotPasswordFormValid(): boolean {
    const resetEmailControl = this.form.get('resetEmail');
    const isValid = !!(resetEmailControl?.value && resetEmailControl?.valid);
    console.log('Debug forgot password form:', {
      value: resetEmailControl?.value,
      valid: resetEmailControl?.valid,
      errors: resetEmailControl?.errors,
      authMode: this.authMode,
      overall: isValid
    });
    return isValid;
  }
}
