# Human — Brand Guidelines

## Brand Essence

**One line:** The programming language that speaks your language.

**Core tension:** Human sits at the intersection of warmth and precision — a compiler that feels like a conversation.

**Personality:** The brand should feel like talking to the smartest, kindest person in the room. Someone who listens carefully, never condescends, and makes complex things feel simple. Not flashy. Not trying to impress. Just quietly, confidently excellent.

---

## Brand Values

**Clarity** — Everything we make should be immediately understandable. If someone has to re-read it, we failed.

**Warmth** — Technology built for people, not the other way around. Approachable without being patronizing.

**Honesty** — We don't overpromise. We show what works. We're transparent about what's in progress.

**Craftsmanship** — Every detail matters. Our compiler enforces quality; our brand should too.

---

## Logo

### Primary Logo — Wordmark

The logo is the word **human** in lowercase, set in a warm, rounded sans-serif typeface, followed by a blinking cursor underscore.

```
human_
```

The underscore is the accent color. It represents:
- A command line waiting for your input
- An invitation — "it's your turn to speak"
- The bridge between thinking and building

The underscore blinks in digital contexts (web, presentations). It's static in print.

### Logo Variations

| Variant | Usage |
|---------|-------|
| `human_` | Primary — wordmark with colored underscore |
| `h_` | Compact — for favicons, app icons, social avatars |
| `human` | Text-only — when the underscore doesn't render well |

### Logo Rules

- Always lowercase. Never "Human" or "HUMAN" in the logo (even though the language keywords are case-insensitive).
- The underscore is always the accent color, never the same color as the text.
- Minimum clear space around the logo: the width of the "h" character on all sides.
- Never stretch, rotate, add effects, or change the typeface.
- On dark backgrounds, text is white, underscore stays accent color.
- On light backgrounds, text is near-black (#1A1A1A), underscore stays accent color.

---

## Color

### Primary Palette

| Name | Hex | Usage |
|------|-----|-------|
| **Ink** | `#1A1A1A` | Primary text, headings, logo on light backgrounds |
| **Paper** | `#FAFAF8` | Page backgrounds, cards |
| **Accent** | `#E85D3A` | The underscore, buttons, links, highlights, CTAs |

The accent is a warm coral-red — energetic but not aggressive. It's the single pop of color in an otherwise monochrome world.

### Extended Palette

| Name | Hex | Usage |
|------|-----|-------|
| **Soft Gray** | `#F0F0EC` | Borders, code block backgrounds, subtle dividers |
| **Medium Gray** | `#8C8C8C` | Secondary text, captions, muted UI elements |
| **Dark** | `#0D0D0D` | Dark mode backgrounds |
| **Accent Light** | `#FFF0EC` | Accent tint for hover states, notifications |
| **Accent Dark** | `#C44A2D` | Accent for dark mode, pressed states |
| **Success** | `#2D8C5A` | Test passing, build success, positive states |
| **Error** | `#C43030` | Build failures, validation errors |
| **Warning** | `#D4940A` | Warnings, deprecation notices |

### Color Rules

- The design is predominantly monochrome. Color is used sparingly and intentionally.
- Accent color appears in: the underscore, primary CTAs, links, selected states, and important highlights.
- Never use accent as a background color for large areas.
- In dark mode, Paper becomes `#0D0D0D`, Ink becomes `#F5F5F3`, Accent stays the same.

---

## Typography

### Typefaces

| Role | Typeface | Fallback |
|------|----------|----------|
| **Logo** | Nunito (Bold, 700) | Rounded sans-serif |
| **Headings** | Source Serif 4 (Semibold, 600) | Georgia, serif |
| **Body** | Nunito Sans (Regular, 400) | system-ui, sans-serif |
| **Code** | JetBrains Mono (Regular, 400) | monospace |

**Why this pairing:**
- **Nunito** for the logo — rounded, warm, friendly, feels human
- **Source Serif 4** for headings — editorial quality, gives authority and seriousness to the content without being cold
- **Nunito Sans** for body — clean, readable, pairs naturally with the logo typeface
- **JetBrains Mono** for code — the standard for developer tools, legible at all sizes

### Type Scale

| Element | Size | Weight | Line Height |
|---------|------|--------|-------------|
| Hero title | 56px / 3.5rem | Semibold | 1.1 |
| H1 | 40px / 2.5rem | Semibold | 1.2 |
| H2 | 32px / 2rem | Semibold | 1.25 |
| H3 | 24px / 1.5rem | Semibold | 1.3 |
| Body large | 20px / 1.25rem | Regular | 1.6 |
| Body | 17px / 1.0625rem | Regular | 1.65 |
| Body small | 14px / 0.875rem | Regular | 1.5 |
| Code | 15px / 0.9375rem | Regular | 1.6 |
| Caption | 13px / 0.8125rem | Regular | 1.4 |

---

## Voice & Tone

### Writing Principles

**Lead with what, not how.** Say "Write English, get a running app" not "Our multi-phase compiler transforms natural language tokens through an AST into framework-agnostic IR."

**Be direct.** Short sentences. Active voice. No hedging. "Human compiles English into code" not "Human is designed to potentially help with the compilation of English-like syntax."

**Be warm, not cute.** Friendly but never silly. We're building a compiler, not a children's app. Think Notion's voice — approachable, clear, professional.

**Show, don't tell.** Always lead with a code example. Let the `.human` code speak for itself. It's the most powerful proof of the concept.

**Respect the reader.** Never condescend. The person reading might be a senior engineer or a first-time founder. Write for both.

### Tone by Context

| Context | Tone |
|---------|------|
| Landing page | Confident, inviting, slightly bold |
| Documentation | Clear, helpful, patient |
| Error messages | Friendly, specific, constructive |
| Social media | Casual, clever, brief |
| Technical blog | Thoughtful, precise, honest |
| README | Practical, concise, welcoming |

### Words We Use

| Instead of... | We say... |
|---------------|-----------|
| Utilize | Use |
| Leverage | Use |
| Empower | Let you / help you |
| Cutting-edge | Modern |
| Revolutionary | Simple / practical |
| Seamless | Easy / smooth |
| Robust | Reliable |
| Best-in-class | Good / effective |

---

## Imagery & Visual Language

### Code as Hero

The `.human` code itself is the primary visual. It's more compelling than any illustration because it proves the concept instantly. Always show real, working examples — never fake or simplified snippets.

### Photography

If photography is ever used, it should feel: natural, warm-lit, slightly desaturated. People thinking, collaborating, or focused. Never stock-photo-perfect. Think editorial photography, not corporate.

### Illustrations

Minimal use. If needed, line-art style with monochrome + accent color. Never 3D renders, gradients blobs, or AI-generated art. Think technical diagrams drawn by hand.

### Icons

Line icons, 1.5px stroke, rounded caps, matching the warmth of the typography. Monochrome with accent for active/selected states.

---

## Components & Patterns

### Buttons

| Type | Style |
|------|-------|
| Primary | Accent background, white text, rounded corners (8px), subtle shadow |
| Secondary | Transparent, Ink border, Ink text, rounded corners (8px) |
| Ghost | No border, accent text, underline on hover |

### Code Blocks

- Background: Soft Gray (`#F0F0EC`) in light mode, `#1A1A1A` in dark mode
- Font: JetBrains Mono
- Border: 1px Soft Gray, rounded (8px)
- Human syntax highlighting: keywords in accent, strings in Success green, comments in Medium Gray

### Cards

- Background: Paper
- Border: 1px Soft Gray
- Border radius: 12px
- Shadow: subtle, warm (0 2px 8px rgba(0,0,0,0.06))
- Padding: generous (24-32px)

---

## Spacing System

Base unit: 4px

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | Tight gaps between related elements |
| sm | 8px | Inline spacing, icon gaps |
| md | 16px | Standard padding, form spacing |
| lg | 24px | Card padding, section gaps |
| xl | 32px | Between content blocks |
| 2xl | 48px | Between major sections |
| 3xl | 64px | Page section separation |
| 4xl | 96px | Hero spacing, major landmarks |

---

## Brand Applications

### Favicon
The `h_` compact mark, accent-colored underscore, on transparent background. Sizes: 16x16, 32x32, 180x180 (Apple touch icon).

### Social Preview (Open Graph)
Dark background (`#0D0D0D`), the `human_` wordmark centered, with a one-line code example below in JetBrains Mono. Accent underscore glowing subtly.

### Terminal / CLI
When running `human build`, output uses: white for standard text, accent for success markers and the `human` brand name, green for pass, red for fail, gray for secondary info.

### GitHub
Repository social preview: dark background, `human_` logo, tagline "The first programming language designed for humans, not computers." below in body type.

---

## Do's and Don'ts

### Do
- Let the code examples be the hero
- Use generous whitespace
- Keep layouts clean and scannable
- Use accent color sparingly for maximum impact
- Show real working examples

### Don't
- Use gradients, glowing effects, or 3D elements
- Use more than one accent color at a time
- Make the design feel "techy" or intimidating
- Use jargon in public-facing copy
- Clutter layouts with too many elements
- Use dark patterns or manipulative CTAs

---

*The Human brand is an extension of the language's philosophy: say what you mean, clearly and warmly, and let the work speak for itself.*
