import { Injectable, signal } from '@angular/core';

type ToastType = 'success' | 'info' | 'warning' | 'error';

export interface ToastOptions {
  type?: ToastType;
  durationMs?: number; // auto-hide delay; set to 0 to keep open until action/hide
  actionLabel?: string;
  action?: () => void;
}

@Injectable({ providedIn: 'root' })
export class ToastService {
  visible = signal(false);
  text = signal('');
  type = signal<ToastType>('info');
  actionLabel = signal<string | null>(null);
  private actionCb: (() => void) | null = null;
  private timer: any = null;

  show(message: string, opts: ToastOptions = {}): void {
    // Clear any pending auto-hide timers
    if (this.timer) { clearTimeout(this.timer); this.timer = null; }

    this.text.set(message);
    this.type.set(opts.type ?? 'info');
    this.actionLabel.set(opts.actionLabel ?? null);
    this.actionCb = opts.action ?? null;
    this.visible.set(true);

    const dur = opts.durationMs ?? (this.actionCb ? 6000 : 3000);
    if (dur > 0) {
      this.timer = setTimeout(() => this.hide(), dur);
    }
  }

  hide(): void {
    if (this.timer) { clearTimeout(this.timer); this.timer = null; }
    this.visible.set(false);
    this.text.set('');
    this.actionLabel.set(null);
    this.actionCb = null;
  }

  triggerAction(): void {
    const cb = this.actionCb;
    this.hide();
    try { cb && cb(); } catch {}
  }
}
