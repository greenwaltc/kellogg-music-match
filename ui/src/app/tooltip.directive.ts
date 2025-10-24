import { Directive, ElementRef, Input, OnDestroy, OnInit } from '@angular/core';

@Directive({
  selector: '[appTooltip]',
  standalone: true,
  host: {
    'tabindex': '0',
    '(mouseenter)': 'show()',
    '(mouseleave)': 'hide()',
    '(focus)': 'show()',
    '(blur)': 'hide()'
  }
})
export class TooltipDirective implements OnInit, OnDestroy {
  @Input('appTooltip') text: string = '';
  @Input() tooltipWidth: string = '220px';
  private el: HTMLElement;
  private tooltipEl?: HTMLElement;
  private static styleInjected = false;
  private repositionHandler = () => this.position();

  constructor(ref: ElementRef<HTMLElement>) {
    this.el = ref.nativeElement;
    if (!TooltipDirective.styleInjected) {
      const style = document.createElement('style');
      style.textContent = `
        .affyne-tooltip { position: fixed; z-index: 9999; background:#1e2530; color:#fff; padding:.45rem .6rem; font-size:.65rem; border-radius:4px; box-shadow:0 2px 8px rgba(0,0,0,.25); line-height:1.25; pointer-events:none; opacity:0; transform:translateY(-4px); transition:opacity .12s ease, transform .12s ease; }
        .affyne-tooltip.visible { opacity:1; transform:translateY(0); }
        .affyne-tooltip[data-pos='top']::after, .affyne-tooltip[data-pos='bottom']::after { content:""; position:absolute; left:18px; border:6px solid transparent; }
        .affyne-tooltip[data-pos='top']::after { bottom:-12px; border-top-color:#1e2530; }
        .affyne-tooltip[data-pos='bottom']::after { top:-12px; border-bottom-color:#1e2530; }
      `;
      document.head.appendChild(style);
      TooltipDirective.styleInjected = true;
    }
  }

  ngOnInit(): void {
    // noop
  }

  show(): void {
    if (!this.text) return;
    if (!this.tooltipEl) {
      this.tooltipEl = document.createElement('div');
      this.tooltipEl.className = 'affyne-tooltip';
      this.tooltipEl.style.width = this.tooltipWidth;
      this.tooltipEl.innerText = this.text;
      document.body.appendChild(this.tooltipEl);
      window.addEventListener('scroll', this.repositionHandler, true);
      window.addEventListener('resize', this.repositionHandler, true);
    }
    requestAnimationFrame(() => {
      this.position();
      this.tooltipEl!.classList.add('visible');
    });
  }

  hide(): void {
    if (this.tooltipEl) {
      this.tooltipEl.classList.remove('visible');
      this.tooltipEl.remove();
      this.tooltipEl = undefined;
      window.removeEventListener('scroll', this.repositionHandler, true);
      window.removeEventListener('resize', this.repositionHandler, true);
    }
  }

  private position(): void {
    if (!this.tooltipEl) return;
    const rect = this.el.getBoundingClientRect();
    const tt = this.tooltipEl;
    const margin = 8;
    const prefBottomY = rect.bottom + margin;
    const willOverflowBottom = prefBottomY + tt.offsetHeight > window.innerHeight;
    if (willOverflowBottom && rect.top - margin - tt.offsetHeight > 0) {
      // place on top
      tt.dataset.pos = 'top';
      tt.style.top = `${rect.top - margin - tt.offsetHeight}px`;
    } else {
      tt.dataset.pos = 'bottom';
      tt.style.top = `${prefBottomY}px`;
    }
    let left = rect.left;
    if (left + tt.offsetWidth > window.innerWidth - 8) {
      left = window.innerWidth - tt.offsetWidth - 8;
    }
    if (left < 8) left = 8;
    tt.style.left = `${left}px`;
  }

  ngOnDestroy(): void {
    this.hide();
  }
}
