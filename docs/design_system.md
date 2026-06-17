# Design System
## Institution OS
**Version**: 1.0.0

---

## 1. Design Philosophy

**Keywords**: professional, formal, human, calm, clean, institutional, timeless, trustworthy, minimal, premium.

**The feeling**: organized, in control, informed, supported — never overwhelmed.

**Avoid**: flashy effects, glassmorphism, neon colors, continuous animations, bouncing widgets, spinning counters, gradient overload.

---

## 2. Color Palette

```typescript
// tailwind.config.ts
const colors = {
  // Primary — deep institutional blue
  primary: {
    50:  '#EFF6FF',
    100: '#DBEAFE',
    200: '#BFDBFE',
    300: '#93C5FD',
    400: '#60A5FA',
    500: '#3B82F6',
    600: '#2563EB',  // primary default
    700: '#1D4ED8',
    800: '#1E40AF',
    900: '#1E3A8A',
    950: '#172554',
  },
  // Neutral — warm slate for text and backgrounds
  neutral: {
    50:  '#F8FAFC',
    100: '#F1F5F9',
    200: '#E2E8F0',
    300: '#CBD5E1',
    400: '#94A3B8',
    500: '#64748B',
    600: '#475569',
    700: '#334155',
    800: '#1E293B',
    900: '#0F172A',
  },
  // Semantic colors
  success: { light: '#DCFCE7', DEFAULT: '#16A34A', dark: '#15803D' },
  warning: { light: '#FEF9C3', DEFAULT: '#CA8A04', dark: '#A16207' },
  error:   { light: '#FEE2E2', DEFAULT: '#DC2626', dark: '#B91C1C' },
  info:    { light: '#DBEAFE', DEFAULT: '#2563EB', dark: '#1D4ED8' },
}
```

---

## 3. Typography

```typescript
// Font: Inter (system-ui fallback)
fontFamily: {
  sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
  mono: ['JetBrains Mono', 'Consolas', 'monospace'],
}

// Type scale
fontSize: {
  xs:   ['0.75rem',  { lineHeight: '1rem' }],
  sm:   ['0.875rem', { lineHeight: '1.25rem' }],
  base: ['1rem',     { lineHeight: '1.5rem' }],
  lg:   ['1.125rem', { lineHeight: '1.75rem' }],
  xl:   ['1.25rem',  { lineHeight: '1.75rem' }],
  '2xl':['1.5rem',   { lineHeight: '2rem' }],
  '3xl':['1.875rem', { lineHeight: '2.25rem' }],
}
```

**Usage Rules:**
- Page titles: `text-2xl font-semibold text-neutral-900`
- Section headers: `text-lg font-semibold text-neutral-800`
- Body text: `text-sm text-neutral-700`
- Labels: `text-xs font-medium text-neutral-500 uppercase tracking-wide`
- Metadata / captions: `text-xs text-neutral-400`

---

## 4. Spacing & Layout

```
Base unit: 4px (Tailwind default)
Page padding: px-6 py-6 (desktop), px-4 py-4 (mobile)
Card padding: p-6
Section gap: gap-6
Form field gap: space-y-4
Sidebar width: 240px (collapsed: 64px)
Topbar height: 64px
Max content width: 1280px (max-w-7xl)
```

---

## 5. Component Patterns

### 5.1 Stat Card (Dashboard)
```tsx
// Simple, no animation — just clear numbers
<Card>
  <CardContent className="p-6">
    <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">
      Students Present
    </p>
    <p className="mt-2 text-3xl font-semibold text-neutral-900">743</p>
    <p className="mt-1 text-sm text-neutral-500">
      of 850 enrolled · <span className="text-success-DEFAULT">87.4%</span>
    </p>
  </CardContent>
</Card>
```

### 5.2 Status Badges
```tsx
// Variants: success | warning | error | info | neutral
<Badge variant="success">Active</Badge>
<Badge variant="warning">Pending</Badge>
<Badge variant="error">Overdue</Badge>
<Badge variant="neutral">Draft</Badge>
```

### 5.3 Data Table
```tsx
// TanStack Table with consistent header/row styling
// Always: sortable columns, pagination, search input
// Row actions: Edit (pencil icon) | View (eye icon) | Delete (trash, red hover)
// Export button top-right: CSV download
```

### 5.4 Forms
```tsx
// React Hook Form + Zod
// Label above input (never placeholder-as-label)
// Error message below input in text-error-DEFAULT text-xs
// Required fields: asterisk in text-error-DEFAULT
// Submit button: primary, right-aligned
// Cancel: ghost, left of submit
```

---

## 6. Framer Motion Guidelines

### ALLOWED uses:
```typescript
// 1. Page transitions (wrap page root element)
const pageVariants = {
  initial: { opacity: 0, y: 8 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.18, ease: 'easeOut' } },
  exit:    { opacity: 0, y: -8, transition: { duration: 0.12 } }
}

// 2. Modal/dialog entrance
const modalVariants = {
  initial: { opacity: 0, scale: 0.97 },
  animate: { opacity: 1, scale: 1, transition: { duration: 0.15 } },
  exit:    { opacity: 0, scale: 0.97 }
}

// 3. Toast/notification slide-in
const toastVariants = {
  initial: { opacity: 0, x: 40 },
  animate: { opacity: 1, x: 0, transition: { duration: 0.2 } },
  exit:    { opacity: 0, x: 40 }
}

// 4. Sidebar collapse (width transition only — no opacity)
// 5. Success state (single subtle scale: 1 → 1.02 → 1, duration 0.2s)
```

### PROHIBITED uses:
```
✗ Bouncing (spring with bounce > 0)
✗ Rotating elements
✗ Continuous/looping animations
✗ Staggered list item animations (data tables especially)
✗ Animated counters (numbers counting up)
✗ Chart animations beyond simple fade-in
✗ Hover scale > 1.02
✗ Any animation > 300ms duration
```

---

## 7. Sidebar Navigation

```
Structure:
├── Dashboard (role-specific)
├── [Core Module 1]
│   ├── Sub-item
│   └── Sub-item
├── [Core Module 2]
...
└── Settings (bottom)

Active state: bg-primary-50 text-primary-700 border-l-2 border-primary-600
Hover state: bg-neutral-100 text-neutral-700
Icon: 20px, consistent Lucide icons
Label: text-sm font-medium
```

---

## 8. Responsive Breakpoints

```
Mobile (< 768px): single column, bottom nav or hamburger
Tablet (768–1024px): 2-column layouts, collapsed sidebar
Desktop (> 1024px): full sidebar + main content
```

**Parent Portal**: optimized for mobile-first (parents primarily use phones)
**Admin/Teacher**: optimized for desktop

---

## 9. Loading States

```tsx
// Skeleton loader for data tables (not spinners)
<Skeleton className="h-8 w-full" />
<Skeleton className="h-8 w-3/4" />

// Full page loader: centered spinner (shadcn Loader2 icon, animate-spin)
// Button loading: replace button text with Loader2 icon + "Loading..."
// Never block entire UI with overlay spinners for < 500ms operations
```

---

## 10. Empty States

```tsx
// Friendly, actionable empty states
<EmptyState
  icon={<UsersIcon className="w-12 h-12 text-neutral-300" />}
  title="No students yet"
  description="Add your first student by starting the admission process."
  action={<Button>Begin Admission</Button>}
/>
```
