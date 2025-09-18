import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-similarity-meter',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="similarity-meter">
      <div class="meter-container">
        <div class="meter-bar">
          <div 
            class="meter-fill" 
            [style.width.%]="percentage"
            [class.high-similarity]="score >= 0.7"
            [class.medium-similarity]="score >= 0.4 && score < 0.7"
            [class.low-similarity]="score < 0.4">
          </div>
        </div>
        <div class="meter-labels">
          <span class="score-text">{{ (score * 100) | number:'1.0-1' }}% match</span>
          <span class="overlap-text">{{ overlap }} shared artists</span>
        </div>
      </div>
      <div class="similarity-circle" 
           [class.high-similarity]="score >= 0.7"
           [class.medium-similarity]="score >= 0.4 && score < 0.7"
           [class.low-similarity]="score < 0.4">
        <svg width="50" height="50" viewBox="0 0 50 50">
          <circle 
            cx="25" 
            cy="25" 
            r="20" 
            fill="none" 
            stroke="#e0e0e0" 
            stroke-width="4">
          </circle>
          <circle 
            cx="25" 
            cy="25" 
            r="20" 
            fill="none" 
            [attr.stroke]="circleColor"
            stroke-width="4" 
            stroke-linecap="round"
            [attr.stroke-dasharray]="circumference"
            [attr.stroke-dashoffset]="dashOffset"
            transform="rotate(-90 25 25)">
          </circle>
          <text 
            x="25" 
            y="30" 
            text-anchor="middle" 
            class="circle-text">
            {{ (score * 100) | number:'1.0-0' }}%
          </text>
        </svg>
      </div>
    </div>
  `,
  styles: [`
    .similarity-meter {
      display: flex;
      align-items: center;
      gap: 1rem;
      margin: 0.5rem 0;
    }

    .meter-container {
      flex: 1;
      min-width: 120px;
    }

    .meter-bar {
      width: 100%;
      height: 12px;
      background-color: #e0e0e0;
      border-radius: 6px;
      overflow: hidden;
      margin-bottom: 0.25rem;
    }

    .meter-fill {
      height: 100%;
      border-radius: 6px;
      transition: width 0.8s ease-in-out;
    }

    .meter-fill.high-similarity {
      background: linear-gradient(90deg, #4CAF50, #66BB6A);
    }

    .meter-fill.medium-similarity {
      background: linear-gradient(90deg, #FF9800, #FFB74D);
    }

    .meter-fill.low-similarity {
      background: linear-gradient(90deg, #F44336, #EF5350);
    }

    .meter-labels {
      display: flex;
      justify-content: space-between;
      font-size: 0.75rem;
      color: #666;
    }

    .score-text {
      font-weight: 600;
    }

    .overlap-text {
      color: #888;
    }

    .similarity-circle {
      flex-shrink: 0;
    }

    .similarity-circle svg {
      display: block;
    }

    .circle-text {
      font-size: 8px;
      font-weight: 600;
      fill: #333;
    }

    .similarity-circle.high-similarity circle:last-of-type {
      stroke: #4CAF50;
    }

    .similarity-circle.medium-similarity circle:last-of-type {
      stroke: #FF9800;
    }

    .similarity-circle.low-similarity circle:last-of-type {
      stroke: #F44336;
    }
  `]
})
export class SimilarityMeterComponent {
  @Input() score: number = 0;
  @Input() overlap: number = 0;

  get percentage(): number {
    return Math.round(this.score * 100);
  }

  get circumference(): number {
    return 2 * Math.PI * 20; // radius = 20
  }

  get dashOffset(): number {
    return this.circumference - (this.score * this.circumference);
  }

  get circleColor(): string {
    if (this.score >= 0.7) return '#4CAF50';
    if (this.score >= 0.4) return '#FF9800';
    return '#F44336';
  }
}