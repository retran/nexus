import { cn } from './utils';

/**
 * Glassmorphism Design System
 *
 * Two-layer architecture:
 * Layer 1 (Panel Material): Elements floating directly over background (with backdrop-filter)
 * Layer 2 (On-Panel Material): Interactive elements on top of panels (without backdrop-filter)
 *
 * Two-material system for actions:
 * 1. Glass: For containers, panels, and secondary/equal actions
 * 2. Solid: For single primary CTA
 */

// Base transition for glass elements
export const glassTransitionBase = cn('transition-colors duration-200');

// Layer 1: Panel Material - for elements floating directly over background
// Used by: Main panel, Dropdowns, Toggles (in corner)
export const glassPanelMaterial = cn(
  'border border-black/15 bg-white/50 backdrop-blur-xl dark:border-white/15 dark:bg-black/50',
  glassTransitionBase
);

// Layer 2: On-Panel Material - for interactive elements on top of glass panels
// NO backdrop-filter to avoid double-blur effect
export const glassOnPanelMaterial = cn(
  'border border-black/15 bg-black/5 dark:border-white/15 dark:bg-white/5',
  glassTransitionBase
);

// Base text colors for glass elements
export const glassTextBase = cn('text-black/90 dark:text-white/90');

export const glassTextSecondaryBase = cn('text-black/60 dark:text-white/60');

// Base hover/focus colors for interactive elements
export const glassHoverFocusBase = cn(
  'hover:bg-black/15 focus-visible:bg-black/15 dark:hover:bg-white/15 dark:focus-visible:bg-white/15'
);

// Base hover/focus colors for dropdown items (lighter)
export const glassDropdownHoverFocusBase = cn(
  'hover:bg-black/10 focus:bg-black/10 dark:hover:bg-white/10 dark:focus:bg-white/10'
);

// Adaptive Typography - inverted colors for readability on glass
export const glassText = glassTextBase;

export const glassTextSecondary = glassTextSecondaryBase;

// Primary Action (Solid Button)
export const primaryCTA = cn(
  'h-10 w-full rounded-lg border border-violet-500 bg-violet-500 font-semibold text-white hover:bg-violet-600 hover:text-white dark:border-violet-500 dark:bg-violet-500 dark:hover:bg-violet-600 dark:hover:text-white',
  glassTransitionBase,
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-400/50'
);

// Secondary Actions (Glass Buttons on Panel)
export const secondaryCTA = cn(
  'flex h-10 w-full max-w-[280px] items-center justify-center gap-2 rounded-lg',
  glassOnPanelMaterial,
  glassTextBase,
  'font-semibold',
  glassHoverFocusBase,
  'focus-visible:outline-none'
);

// Tertiary Actions (Text Links)
export const tertiaryAction = cn(
  'text-sm',
  glassTextSecondaryBase,
  'hover:text-black/90 hover:underline focus-visible:text-black/90 focus-visible:underline dark:hover:text-white/90 dark:focus-visible:text-white/90',
  glassTransitionBase,
  'focus-visible:outline-auto'
);

// External Links (for <a> tags)
export const glassLink = cn(
  glassTextSecondaryBase,
  'decoration-1 underline-offset-2 hover:text-black/90 hover:underline focus-visible:text-black/90 focus-visible:underline dark:hover:text-white/90 dark:focus-visible:text-white/90',
  glassTransitionBase
);

// Input Fields (on panel - no backdrop-filter)
export const glassInput = cn(
  'h-10 rounded-lg',
  glassOnPanelMaterial,
  glassTextBase,
  'placeholder:text-black/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500/50 dark:placeholder:text-white/60'
);

// Labels
export const glassLabel = cn('text-sm', glassTextBase);

// Checkbox (on panel - no backdrop-filter)
export const glassCheckbox = cn(
  glassOnPanelMaterial,
  'focus-visible:ring-2 focus-visible:ring-violet-500/50 data-[state=checked]:border-violet-500 data-[state=checked]:bg-violet-500 data-[state=checked]:text-white data-[state=checked]:hover:brightness-90 dark:data-[state=checked]:border-violet-500 dark:data-[state=checked]:bg-violet-500 dark:data-[state=checked]:text-white'
);

// Dropdown Content (floating over background - with backdrop-filter)
export const glassDropdown = cn(glassPanelMaterial, 'rounded-lg shadow-xl');

// Dropdown Item
export const glassDropdownItem = cn(
  glassDropdownHoverFocusBase,
  glassTransitionBase,
  glassTextBase
);

// Glass Panel (main auth panel)
export const glassPanel = cn(
  glassPanelMaterial,
  'w-sm md:w-sm relative z-10 flex h-[35em] flex-col rounded-2xl p-8 shadow-xl md:ml-auto md:mr-[calc(60%-50vw)]'
);

// Toggle Trigger (for theme/language toggles)
export const glassToggleTrigger = cn(
  glassPanelMaterial,
  'relative flex h-9 w-9 items-center justify-center overflow-hidden rounded-lg focus-visible:outline-none',
  glassTransitionBase,
  'before:absolute before:inset-0 before:bg-transparent before:transition-colors before:duration-200 hover:before:bg-black/15 dark:hover:before:bg-white/15'
);

// Separator with text
export const glassSeparator = cn('relative w-full max-w-[280px]');

export const glassSeparatorText = cn('bg-white/10 px-2 dark:bg-black/10');

// Header
export const glassHeader = cn(
  'flex flex-col items-center justify-center space-y-3'
);

export const glassHeaderIcon = cn('h-16 w-16');

export const glassHeaderTitle = cn('text-2xl font-semibold');

// Legacy export for backward compatibility
export const glassBase = glassPanelMaterial;
