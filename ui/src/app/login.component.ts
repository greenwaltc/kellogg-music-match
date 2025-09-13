import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators, FormGroup, FormControl } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from './auth.service';
import { MatchService } from './match.service';
import { passwordComplexityValidator, confirmPasswordValidator, getPasswordStrength, PasswordComplexity } from './password-validators';

interface LoginFormShape { 
  username: FormControl<string | null>; 
  email: FormControl<string | null>; 
  firstName: FormControl<string | null>; 
  lastName: FormControl<string | null>;
  password: FormControl<string | null>; 
  confirmPassword: FormControl<string | null>;
}

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  template: `
  <section class="login-hero fade-in">
    <div class="hero-inner slide-down">
      <h1>Kellogg Music Match</h1>
      <p class="tagline">Connect with classmates who share your music taste.</p>
      
      <div class="auth-toggle">
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

      <form class="auth-form" [formGroup]="form" (ngSubmit)="submit()" novalidate>
        <h2>{{ isRegisterMode ? 'Create Account' : 'Sign In' }}</h2>
        
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
        
        <div class="error form-error" *ngIf="auth.error()">{{ auth.error() }}</div>
      </form>
      
      <p class="legal">By continuing you agree to participate in music matching.</p>
    </div>
  </section>
  `,
  styles: [
    `:host { display:block; }
  .login-hero { min-height: calc(100vh - 160px); display:flex; align-items:center; justify-content:center; padding:2rem 1rem; }
  .hero-inner { width:100%; max-width:780px; margin:0 auto; text-align:center; }
  .hero-inner h1 { font-size:2.6rem; margin:0 0 0.85rem; background:var(--color-gradient); -webkit-background-clip:text; color:transparent; }
  .tagline { color:var(--color-text-muted); margin:0 0 1.9rem; font-size:1.05rem; }
  
  .auth-toggle { display:flex; margin:0 auto 1.5rem; width:100%; max-width:460px; background:var(--color-surface); border:1px solid var(--color-border); border-radius:12px; padding:4px; }
  .auth-toggle button { flex:1; padding:0.75rem 1rem; background:transparent; border:none; color:var(--color-text-muted); font-size:0.9rem; font-weight:500; border-radius:8px; cursor:pointer; transition:all 0.2s ease; }
  .auth-toggle button.active { background:var(--color-accent); color:white; }
  .auth-toggle button:hover:not(.active) { color:var(--color-text); }
  
  .auth-form { backdrop-filter: blur(14px); background:var(--color-surface-translucent); border:1px solid var(--color-border); padding:2.15rem 1.9rem 2.4rem; border-radius:24px; margin:0 auto; width:100%; max-width:460px; text-align:left; box-shadow:0 10px 36px -8px rgba(0,0,0,0.55), 0 0 0 1px rgba(255,255,255,0.02) inset; animation: float-in .7s cubic-bezier(.3,.7,.4,1); }
  .auth-form h2 { margin:0 0 1.25rem; font-size:1.05rem; letter-spacing:0.11em; text-transform:uppercase; font-weight:600; color:var(--color-text-muted); }
    .field { margin-bottom:1rem; }
    .field input { width:100%; }
    
  /* Name fields side by side */
  .name-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-bottom: 1rem; }
  .name-fields .field { margin-bottom: 0; }
  
  /* Password input with toggle */
  .password-input-container { position: relative; }
  .password-input-container input { padding-right: 4rem; }
  .password-toggle { position: absolute; right: 0.75rem; top: 50%; transform: translateY(-50%); background: none; border: none; cursor: pointer; font-size: 0.8rem; color: var(--color-text-muted); transition: color 0.2s ease; font-weight: 500; }
  .password-toggle:hover { color: var(--color-text); }
  .password-toggle:focus { outline: none; color: var(--color-accent); }
  
  /* Password requirements styling */
  .password-requirements { margin-top: 0.75rem; padding: 0.75rem; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 8px; font-size: 0.85rem; }
  .requirement-header { font-weight: 600; color: var(--color-text); margin-bottom: 0.5rem; }
  .requirements-list { display: flex; flex-direction: column; gap: 0.25rem; }
  .requirement { display: flex; align-items: center; gap: 0.5rem; color: var(--color-text-muted); transition: color 0.2s ease; }
  .requirement.met { color: var(--color-accent); }
  .requirement-icon { font-weight: bold; width: 1rem; text-align: center; }
  
  /* Password strength indicator */
  .password-strength { margin-top: 0.75rem; }
  .strength-bar { height: 4px; background: var(--color-border); border-radius: 2px; overflow: hidden; margin-bottom: 0.25rem; }
  .strength-fill { height: 100%; transition: width 0.3s ease, background-color 0.3s ease; border-radius: 2px; }
  .strength-label { font-size: 0.75rem; font-weight: 600; }
  
  .primary-btn { width:100%; justify-content:center; background:linear-gradient(135deg,var(--color-accent),var(--color-accent-alt)); box-shadow:0 4px 18px -6px rgba(0,0,0,0.5); transition: transform .25s ease, box-shadow .25s ease; }
  .primary-btn:hover:not([disabled]) { transform:translateY(-2px); box-shadow:0 8px 24px -8px rgba(0,0,0,0.55); }
  .primary-btn:active { transform:translateY(0); }
  .primary-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }
  .legal { margin-top:1.25rem; font-size:0.7rem; color:var(--color-text-muted); letter-spacing:0.05em; }
  .fade-in { animation: fade-in .8s ease; }
  .slide-down { animation: slide-down .7s cubic-bezier(.25,.7,.3,1); }
  @keyframes fade-in { from { opacity:0; } to { opacity:1; } }
  @keyframes slide-down { from { transform:translateY(-18px); opacity:0; } to { transform:translateY(0); opacity:1; } }
  @keyframes float-in { from { transform:translateY(18px) scale(.98); opacity:0; } to { transform:translateY(0) scale(1); opacity:1; } }
    .form-error { margin-top:0.75rem; text-align:center; }
    @media (max-width:600px){ 
      .hero-inner h1 { font-size:1.9rem; } 
      .auth-form { padding:1.5rem 1.25rem 1.75rem; } 
      .name-fields { grid-template-columns: 1fr; gap: 0; }
      .name-fields .field { margin-bottom: 1rem; }
    }
    `
  ]
})
export class LoginComponent {
  form!: FormGroup<LoginFormShape>;
  isRegisterMode = false;
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

  constructor(private fb: FormBuilder, public auth: AuthService, private router: Router, private matches: MatchService) {
    this.initializeForm();
  }

  private initializeForm(): void {
    this.form = this.fb.group<LoginFormShape>({
      username: this.fb.control<string | null>('', [Validators.required, Validators.minLength(3)]),
      email: this.fb.control<string | null>('', [Validators.required, Validators.email]),
      firstName: this.fb.control<string | null>('', [Validators.required, Validators.minLength(2)]),
      lastName: this.fb.control<string | null>('', [Validators.required, Validators.minLength(2)]),
      password: this.fb.control<string | null>('', [Validators.required, Validators.minLength(6)]),
      confirmPassword: this.fb.control<string | null>('', [Validators.required])
    });
    this.updateValidators();
  }

  toggleMode(isRegister: boolean): void {
    this.isRegisterMode = isRegister;
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
    const passwordControl = this.form.get('password');
    const confirmPasswordControl = this.form.get('confirmPassword');
    
    if (this.isRegisterMode) {
      emailControl?.setValidators([Validators.required, Validators.email]);
      firstNameControl?.setValidators([Validators.required, Validators.minLength(2)]);
      lastNameControl?.setValidators([Validators.required, Validators.minLength(2)]);
      passwordControl?.setValidators([Validators.required, passwordComplexityValidator()]);
      confirmPasswordControl?.setValidators([Validators.required, confirmPasswordValidator('password')]);
    } else {
      emailControl?.clearValidators();
      firstNameControl?.clearValidators();
      lastNameControl?.clearValidators();
      passwordControl?.setValidators([Validators.required, Validators.minLength(6)]);
      confirmPasswordControl?.clearValidators();
    }
    
    emailControl?.updateValueAndValidity();
    firstNameControl?.updateValueAndValidity();
    lastNameControl?.updateValueAndValidity();
    passwordControl?.updateValueAndValidity();
    confirmPasswordControl?.updateValueAndValidity();
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
    
    const { username, email, firstName, lastName, password } = this.form.value;
    
    if (this.isRegisterMode) {
      const registerData = {
        username: username || '',
        email: email || '',
        firstName: firstName || '',
        lastName: lastName || '',
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
}
