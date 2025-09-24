import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, NgForm } from '@angular/forms';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { AuthService } from './auth.service';
import { environment } from '../environments/environment';

@Component({
  selector: 'app-feedback',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="feedback-container">
      <h2>Share Your Feedback</h2>
      <p>We'd love to hear your thoughts and suggestions to improve the Music Match experience!</p>
      
      <form (ngSubmit)="submitFeedback(feedbackForm)" #feedbackForm="ngForm">
        <div class="form-group">
          <label for="feedback">Your Feedback:</label>
          <textarea 
            id="feedback"
            name="feedback"
            [(ngModel)]="feedbackText"
            [maxlength]="maxLength"
            rows="8"
            placeholder="Share your thoughts, suggestions, or report any issues..."
            required
            #feedbackTextarea="ngModel">
          </textarea>
          <div class="character-count">
            {{ feedbackText.length }} / {{ maxLength }} characters
          </div>
          <div *ngIf="feedbackTextarea.invalid && feedbackTextarea.touched" class="error">
            Feedback is required
          </div>
        </div>
        
        <div class="form-actions">
          <button 
            type="submit" 
            [disabled]="feedbackForm.invalid || isSubmitting || feedbackText.trim().length === 0"
            class="submit-btn">
            {{ isSubmitting ? 'Submitting...' : 'Submit Feedback' }}
          </button>
        </div>
      </form>
      
      <div *ngIf="successMessage" class="success-message">
        {{ successMessage }}
      </div>
      
      <div *ngIf="errorMessage" class="error-message">
        {{ errorMessage }}
      </div>
    </div>
  `,
  styles: [`
    .feedback-container {
      max-width: 600px;
      margin: 0 auto;
      padding: 2rem;
    }

    h2 {
      color: var(--color-accent);
      margin-bottom: 0.5rem;
    }

    p {
      color: var(--color-text-muted);
      margin-bottom: 2rem;
      line-height: 1.6;
    }

    .form-group {
      margin-bottom: 1.5rem;
    }

    label {
      display: block;
      margin-bottom: 0.5rem;
      font-weight: 500;
      color: var(--color-text);
    }

    textarea {
      width: 100%;
      padding: 0.75rem;
      border: 1px solid var(--color-border);
      border-radius: 4px;
      background-color: var(--color-input-bg);
      color: var(--color-text);
      font-family: inherit;
      font-size: 1rem;
      line-height: 1.4;
      resize: vertical;
      min-height: 120px;
    }

    textarea:focus {
      outline: none;
      border-color: var(--color-accent);
      box-shadow: 0 0 0 2px rgba(106, 42, 185, 0.2);
    }

    .character-count {
      text-align: right;
      font-size: 0.875rem;
      color: var(--color-text-muted);
      margin-top: 0.25rem;
    }

    .form-actions {
      display: flex;
      justify-content: flex-end;
    }

    .submit-btn {
      background-color: var(--color-accent);
      color: white;
      border: none;
      padding: 0.75rem 1.5rem;
      border-radius: 4px;
      font-size: 1rem;
      cursor: pointer;
      transition: background-color 0.2s ease;
    }

    .submit-btn:hover:not(:disabled) {
      background-color: var(--color-accent-alt);
    }

    .submit-btn:disabled {
      background-color: var(--color-text-muted);
      cursor: not-allowed;
    }

    .success-message {
      background-color: rgba(34, 197, 94, 0.1);
      color: #22c55e;
      padding: 1rem;
      border-radius: 4px;
      border: 1px solid #22c55e;
      margin-top: 1rem;
    }

    .error-message {
      background-color: rgba(248, 113, 113, 0.1);
      color: var(--color-error);
      padding: 1rem;
      border-radius: 4px;
      border: 1px solid var(--color-error);
      margin-top: 1rem;
    }

    .error {
      color: var(--color-error);
      font-size: 0.875rem;
      margin-top: 0.25rem;
    }

    @media (max-width: 768px) {
      .feedback-container {
        padding: 1rem;
      }
      
      h2 {
        font-size: 1.5rem;
      }
    }
  `]
})
export class FeedbackComponent {
  private http = inject(HttpClient);
  private authService = inject(AuthService);

  feedbackText = '';
  maxLength = 1000;
  isSubmitting = false;
  successMessage = '';
  errorMessage = '';

  async submitFeedback(form: NgForm) {
    if (this.feedbackText.trim().length === 0) {
      this.errorMessage = 'Please enter your feedback before submitting.';
      return;
    }

    this.isSubmitting = true;
    this.successMessage = '';
    this.errorMessage = '';

    try {
      const currentUser = this.authService.user();
      if (!currentUser?.username) {
        throw new Error('Please log in to submit feedback');
      }

      const headers = new HttpHeaders({
        'Content-Type': 'application/json',
        'X-User-Username': currentUser.username
      });

      const response = await this.http.post(
        `${environment.apiBaseUrl}/feedback`,
        { feedback: this.feedbackText.trim() },
        { headers }
      ).toPromise();

      this.successMessage = 'Thank you for your feedback! We appreciate your input.';
      this.feedbackText = '';
      this.errorMessage = '';
      
      // Reset form state to pristine/untouched to clear validation errors
      form.resetForm({ feedback: '' });
    } catch (error: any) {
      console.error('Error submitting feedback:', error);
      this.errorMessage = error.error?.message || 'Failed to submit feedback. Please try again.';
      this.successMessage = '';
    } finally {
      this.isSubmitting = false;
    }
  }
}