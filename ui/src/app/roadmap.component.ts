import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'app-roadmap',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="roadmap-container">
      <h2>🗺️ Roadmap</h2>
      <p class="intro">
        If this app gets good momentum and there's enough interest, I have several ideas for future features and changes. 
        If you have suggestions, please submit feedback!
      </p>
      
      <div class="roadmap-content">
        <ul class="feature-list">
          <li class="feature-item">
            <div class="feature-icon">🎵</div>
            <div class="feature-text">
              <strong>Music Platform Integrations</strong>
              <p>Integrations with Spotify, Apple Music, etc so that music preferences are automatically registered.</p>
            </div>
          </li>
          
          <li class="feature-item">
            <div class="feature-icon">🧠</div>
            <div class="feature-text">
              <strong>Advanced Similarity Algorithms</strong>
              <p>More powerful similarity comparison algorithms, e.g., similar artists, individual songs, some sort of fancy deep learning vectorization and latent space distancing, etc.</p>
            </div>
          </li>
          
          <li class="feature-item">
            <div class="feature-icon">🤝</div>
            <div class="feature-text">
              <strong>Connection Facilitation</strong>
              <p>Some sort of mechanism to facilitate connecting with your matches.</p>
            </div>
          </li>
          
          <li class="feature-item">
            <div class="feature-icon">🌟</div>
            <div class="feature-text">
              <strong>Beyond Music</strong>
              <p>Expanding to interests beyond music.</p>
            </div>
          </li>
          
          <li class="feature-item">
            <div class="feature-icon">💡</div>
            <div class="feature-text">
              <strong>Your Ideas</strong>
              <p>Any of your suggestions!</p>
            </div>
          </li>
        </ul>
        
        <div class="cta-section">
          <p>Have ideas or feedback? We'd love to hear from you!</p>
          <a routerLink="/feedback" class="cta-button">Submit Feedback</a>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .roadmap-container {
      max-width: 800px;
      margin: 0 auto;
      padding: 2rem;
    }

    h2 {
      color: var(--color-accent);
      margin-bottom: 0.5rem;
      font-size: 2rem;
      text-align: center;
    }

    .intro {
      color: var(--color-text-muted);
      margin-bottom: 2rem;
      line-height: 1.6;
      text-align: center;
      font-size: 1.1rem;
    }

    .roadmap-content {
      background: var(--color-card-bg, #ffffff);
      border: 1px solid var(--color-border);
      border-radius: 12px;
      padding: 2rem;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    .feature-list {
      list-style: none;
      padding: 0;
      margin: 0;
    }

    .feature-item {
      display: flex;
      align-items: flex-start;
      gap: 1rem;
      padding: 1.5rem 0;
      border-bottom: 1px solid var(--color-border);
    }

    .feature-item:last-child {
      border-bottom: none;
    }

    .feature-icon {
      font-size: 1.5rem;
      min-width: 2rem;
      text-align: center;
      margin-top: 0.25rem;
    }

    .feature-text {
      flex: 1;
    }

    .feature-text strong {
      color: var(--color-text);
      font-size: 1.1rem;
      display: block;
      margin-bottom: 0.5rem;
    }

    .feature-text p {
      color: var(--color-text-muted);
      margin: 0;
      line-height: 1.5;
    }

    .cta-section {
      margin-top: 2rem;
      padding-top: 2rem;
      border-top: 1px solid var(--color-border);
      text-align: center;
    }

    .cta-section p {
      color: var(--color-text-muted);
      margin-bottom: 1rem;
      font-size: 1.1rem;
    }

    .cta-button {
      display: inline-block;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      text-decoration: none;
      padding: 0.75rem 1.5rem;
      border-radius: 8px;
      font-weight: 500;
      transition: transform 0.2s ease, box-shadow 0.2s ease;
    }

    .cta-button:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
    }

    .cta-button:active {
      transform: translateY(0);
    }

    /* Dark mode support */
    @media (prefers-color-scheme: dark) {
      .roadmap-content {
        background: var(--color-card-bg, #2a2a2a);
      }
    }

    /* Responsive design */
    @media (max-width: 768px) {
      .roadmap-container {
        padding: 1rem;
      }

      .roadmap-content {
        padding: 1.5rem;
      }

      .feature-item {
        flex-direction: column;
        gap: 0.5rem;
        padding: 1rem 0;
      }

      .feature-icon {
        align-self: flex-start;
      }

      h2 {
        font-size: 1.5rem;
      }
    }
  `]
})
export class RoadmapComponent {
  constructor() {}
}