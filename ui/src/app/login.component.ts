import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators, FormGroup, FormControl } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from './auth.service';
import { MatchService } from './match.service';

interface LoginFormShape { email: FormControl<string | null>; fullName: FormControl<string | null>; }

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  template: `
  <section class="login-hero fade-in">
    <div class="hero-inner slide-down">
  <h1>Kellogg Music Match</h1>
      <p class="tagline">Connect with classmates who share your music taste.</p>
      <form class="auth-form" [formGroup]="form" (ngSubmit)="submit()" novalidate>
        <h2>Login / Register</h2>
        <div class="field">
          <label for="email">Email</label>
          <input id="email" type="email" formControlName="email" placeholder="Kellogg Student Email" />
          <div class="error" *ngIf="form.get('email')?.touched && form.get('email')?.invalid">Valid email required</div>
        </div>
        <div class="field">
          <label for="fullName">Full Name</label>
          <input id="fullName" type="text" formControlName="fullName" placeholder="Full Name" />
          <div class="error" *ngIf="form.get('fullName')?.touched && form.get('fullName')?.invalid">Name required (min 2 chars)</div>
        </div>
  <button type="submit" class="primary-btn" [disabled]="auth.loading()">{{ auth.loading() ? 'Submitting...' : 'Continue' }}</button>
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
  .auth-form { backdrop-filter: blur(14px); background:var(--color-surface-translucent); border:1px solid var(--color-border); padding:2.15rem 1.9rem 2.4rem; border-radius:24px; margin:0 auto; width:100%; max-width:460px; text-align:left; box-shadow:0 10px 36px -8px rgba(0,0,0,0.55), 0 0 0 1px rgba(255,255,255,0.02) inset; animation: float-in .7s cubic-bezier(.3,.7,.4,1); }
  .auth-form h2 { margin:0 0 1.25rem; font-size:1.05rem; letter-spacing:0.11em; text-transform:uppercase; font-weight:600; color:var(--color-text-muted); }
    .field { margin-bottom:1rem; }
    .field input { width:100%; }
  .primary-btn { width:100%; justify-content:center; background:linear-gradient(135deg,var(--color-accent),var(--color-accent-alt)); box-shadow:0 4px 18px -6px rgba(0,0,0,0.5); transition: transform .25s ease, box-shadow .25s ease; }
  .primary-btn:hover:not([disabled]) { transform:translateY(-2px); box-shadow:0 8px 24px -8px rgba(0,0,0,0.55); }
  .primary-btn:active { transform:translateY(0); }
  .legal { margin-top:1.25rem; font-size:0.7rem; color:var(--color-text-muted); letter-spacing:0.05em; }
  .fade-in { animation: fade-in .8s ease; }
  .slide-down { animation: slide-down .7s cubic-bezier(.25,.7,.3,1); }
  @keyframes fade-in { from { opacity:0; } to { opacity:1; } }
  @keyframes slide-down { from { transform:translateY(-18px); opacity:0; } to { transform:translateY(0); opacity:1; } }
  @keyframes float-in { from { transform:translateY(18px) scale(.98); opacity:0; } to { transform:translateY(0) scale(1); opacity:1; } }
    .form-error { margin-top:0.75rem; text-align:center; }
    @media (max-width:600px){ .hero-inner h1 { font-size:1.9rem; } .auth-form { padding:1.5rem 1.25rem 1.75rem; } }
    `
  ]
})
export class LoginComponent {
  form!: FormGroup<LoginFormShape>;

  constructor(private fb: FormBuilder, public auth: AuthService, private router: Router, private matches: MatchService) {
    this.form = this.fb.group<LoginFormShape>({
      email: this.fb.control<string | null>('', [Validators.required, Validators.email]),
      fullName: this.fb.control<string | null>('', [Validators.required, Validators.minLength(2)])
    });
  }

  submit(): void {
    if (this.form.invalid) { this.form.markAllAsTouched(); return; }
    const { email, fullName } = this.form.value;
    this.auth.login(email || '', fullName || '').subscribe({
      next: () => {
        this.matches.fetchIfReady();
        const hasArtists = !!this.auth.user()?.artists?.length;
        this.router.navigateByUrl(hasArtists ? '/matches' : '/artists');
      }
    });
  }
}
