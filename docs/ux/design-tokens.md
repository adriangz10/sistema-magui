# Design tokens — Sistema Magui (rediseño minimalista)

Estado: propuesta para aprobación. Reemplaza los tokens actuales de `frontend/src/styles.css`
(rosa `brand` + grises azulados `ink` + verde/ámbar/rojo saturados).
Stack: Tailwind CSS 4 (`@theme`), tema claro/oscuro por clase `dark` en `<html>` (sin cambios).

## 1. Principios

1. **Neutro primero.** Todo es gris cálido (familia "stone"). El color se reserva para informar: error, stock bajo, por vencer.
2. **Un solo acento sobrio** (`brand`, azul-verdoso apagado) para foco, enlaces, elemento seleccionado y gráficos. No se usa en botones principales.
3. **Botón principal monocromo** (casi negro en claro, casi blanco en oscuro): jerarquía por contraste, no por color.
4. **Planos y con bordes finos.** Sin gradientes, sin sombras decorativas, sin `backdrop-blur`. Sombra solo en lo que flota (menú desplegable, modal, toast).
5. **Estados de "todo bien" no se pintan.** "Confirmada", "Vigente", "En línea" van en neutro. Solo lo anómalo lleva color.
6. **Accesible:** AA en todos los pares; foco siempre visible; el color nunca es el único indicador (siempre texto o ícono).

## 2. Paleta

### 2.1 Neutros — `ink` (se mantiene el nombre; cambian los valores a stone)

| Token | Hex | Uso principal |
|---|---|---|
| ink-50 | `#fafaf9` | Fondo de la app (claro), hover de filas |
| ink-100 | `#f5f5f4` | Fondo de chips/badges neutros, ítem de nav activo, hover de botón fantasma |
| ink-200 | `#e7e5e4` | Bordes de cards, header, sidebar, divisores |
| ink-300 | `#d6d3d1` | Borde de inputs y botón secundario |
| ink-400 | `#a8a29e` | Solo decorativo (íconos inertes, ejes de gráfico). **Nunca texto** en claro |
| ink-500 | `#78716c` | Íconos secundarios (no texto chico sobre ink-100) |
| ink-600 | `#57534e` | Texto secundario fuerte |
| ink-700 | `#44403c` | Texto de cuerpo en controles, hover de botón principal |
| ink-800 | `#292524` | Bordes y superficies elevadas (oscuro) |
| ink-900 | `#1c1917` | Texto principal (claro) / superficie de cards (oscuro) / botón principal (claro) |
| ink-950 | `#0c0a09` | Fondo de la app (oscuro) |

### 2.2 Acento único — `brand` (azul-verdoso apagado, saturación baja)

| Token | Hex | Uso |
|---|---|---|
| brand-50 | `#f1f5f5` | Fondo de elemento seleccionado (claro) |
| brand-100 | `#e1eaec` | — |
| brand-200 | `#c5d5d9` | — |
| brand-300 | `#9bb8be` | Texto/foco/enlaces en oscuro |
| brand-400 | `#6f969e` | Barras de gráfico en oscuro |
| brand-500 | `#527b84` | — |
| brand-600 | `#3f6670` | Foco, enlaces, barras de gráfico, checkbox/radio (claro). Contraste 6.3:1 sobre blanco |
| brand-700 | `#33525a` | Hover de enlaces |
| brand-800 | `#2a4349` | — |
| brand-950 | `#131f22` | Fondo de elemento seleccionado (oscuro) |

### 2.3 Semánticos apagados (con variables CSS que cambian solos en oscuro)

| Token | Claro | Oscuro | Uso |
|---|---|---|---|
| `ok` / `ok-soft` / `ok-line` | `#3d6b4a` / `#f1f6f2` / `#cfe0d4` | `#8fc09e` / `#17241c` / `#2a4132` | Confirmaciones puntuales (toast de éxito, "cobro registrado", vuelto) |
| `warn` / `warn-soft` / `warn-line` | `#8a5a14` / `#f8f3ea` / `#e8dcc2` | `#d6b06a` / `#251e12` / `#4a3a1d` | Stock bajo, por vencer, offline, pendientes de sincronizar |
| `danger` / `danger-soft` / `danger-line` | `#a23a3a` / `#f8eeee` / `#e9cccc` | `#e39a9a` / `#2a1818` / `#4d2b2b` | Errores, sin stock, vencido, anulada |
| `muted` | `#6b6560` | `#a8a29e` | **Todo el texto secundario** (reemplaza `text-ink-500`/`text-ink-400` usados como texto) |

Contrastes verificados (≥ 4.5:1): `ok`/`warn`/`danger` sobre blanco y sobre su `-soft` claro (6.2 / 5.9 / 6.5 sobre blanco); versiones oscuras sobre su `-soft` oscuro y sobre ink-900 (>7:1); `muted` sobre ink-50/ink-100/blanco (5.2–5.6) y en oscuro sobre ink-900 (6.9).

> Hallazgo de accesibilidad existente: `text-ink-500` sin variante `dark:` (muy frecuente) da ~3.2:1 sobre fondo oscuro, incumple AA. El token `muted` lo resuelve porque cambia de valor por tema.

### 2.4 Superficies y bordes (guía de uso, sin tokens nuevos)

| Elemento | Claro | Oscuro |
|---|---|---|
| Fondo de la app | `bg-ink-50` | `dark:bg-ink-950` |
| Card, sidebar, header, modal, input | `bg-white` | `dark:bg-ink-900` |
| Borde estándar | `border-ink-200` | `dark:border-ink-800` |
| Borde de control (input, botón secundario) | `border-ink-300` | `dark:border-ink-700` |
| Divisor interno / fila de tabla | `border-ink-100` | `dark:border-ink-800` |
| Texto principal | `text-ink-900` | `dark:text-ink-100` |
| Texto secundario | `text-muted` | (automático) |

## 3. Tipografía

Se mantiene **Inter Variable** (`--font-sans`). Se simplifica a tres pesos: 400, 500, 600 (se elimina `font-bold`).

| Rol | Clases |
|---|---|
| Título de página | `text-xl font-semibold tracking-tight lg:text-2xl` (clase `.titulo-pagina`) |
| Título de sección / card | `text-base font-semibold` |
| Cuerpo | `text-sm` (14px) |
| Texto de apoyo | `text-sm text-muted` |
| Etiqueta de tabla | `text-xs font-medium text-muted` — **sin mayúsculas ni tracking** |
| Cifras (KPI, importes, tablas) | `tabular-nums font-semibold` (KPI: `text-2xl`) |
| Total a cobrar | `text-3xl font-semibold tabular-nums` |

## 4. Espaciado, radios, sombras

- Escala de 4 px de Tailwind. Padding de card: `p-4` (móvil) / `p-5` (≥ sm). Separación entre secciones: `space-y-6`. Tap target mínimo 40 px (`min-h-10`), 44 px en barra inferior y botón Cobrar (`h-12`).
- **Radios:** controles (botón, input, chip, badge) `rounded-lg` (8 px; badge `rounded-md`); card, modal, dropdown, toast `rounded-xl` (12 px); bottom-sheet `rounded-t-2xl`. Se eliminan `rounded-3xl` y `rounded-2xl` en cards. Excepción: avatar/ícono circular `rounded-full`.
- **Sombras:** `shadow-none` en cards, botones, header e inputs (se eliminan `shadow-sm`, `shadow-xs`, `shadow`). Flotantes: `shadow-md` en dropdown del buscador, tooltip del gráfico, toast y modal; bottom-sheet `shadow-md`. Prohibido `shadow-lg/xl/2xl`.
- **Bordes:** 1 px siempre; el grosor 2 px solo para foco y la pestaña activa.

## 5. Estados interactivos

| Estado | Regla |
|---|---|
| Hover | Cambio de fondo a un paso (ej. `hover:bg-ink-100`); sin cambios de sombra ni escala |
| Foco visible | `outline-2 outline-offset-2 outline-brand-600` (`dark:outline-brand-300`) en todo lo interactivo; inputs: `outline-2 -outline-offset-1` |
| Activo/presionado | Un paso más oscuro que hover |
| Disabled | `opacity-50 cursor-not-allowed`; input: fondo `ink-100` / `dark:ink-800` |
| Seleccionado | Fondo `brand-50` (`dark:brand-950`) + borde `brand-600` (`dark:brand-300`); en chips/segmentados: fondo `ink-900` + texto blanco (`dark:` invertido) |
| Error de campo | borde y mensaje en `danger`, mensaje con ícono (no solo color) |
| Transiciones | `transition-colors duration-150`; respetar `motion-reduce:transition-none` |

## 6. Implementación — `frontend/src/styles.css` (reemplazo completo)

```css
@import '@fontsource-variable/inter';
@import 'tailwindcss';

@theme {
  --font-sans: 'Inter Variable', ui-sans-serif, system-ui, sans-serif;

  /* Acento único, apagado */
  --color-brand-50: #f1f5f5;
  --color-brand-100: #e1eaec;
  --color-brand-200: #c5d5d9;
  --color-brand-300: #9bb8be;
  --color-brand-400: #6f969e;
  --color-brand-500: #527b84;
  --color-brand-600: #3f6670;
  --color-brand-700: #33525a;
  --color-brand-800: #2a4349;
  --color-brand-950: #131f22;

  /* Neutros cálidos (stone) */
  --color-ink-50: #fafaf9;
  --color-ink-100: #f5f5f4;
  --color-ink-200: #e7e5e4;
  --color-ink-300: #d6d3d1;
  --color-ink-400: #a8a29e;
  --color-ink-500: #78716c;
  --color-ink-600: #57534e;
  --color-ink-700: #44403c;
  --color-ink-800: #292524;
  --color-ink-900: #1c1917;
  --color-ink-950: #0c0a09;

  /* Sombras mínimas: solo para elementos flotantes */
  --shadow-md: 0 4px 12px -2px rgb(28 25 23 / 0.10), 0 2px 4px -2px rgb(28 25 23 / 0.06);
}

/* Semánticos: cambian solos con el tema (evita repetir dark:text-green-300, etc.) */
@theme inline {
  --color-ok: var(--c-ok);
  --color-ok-soft: var(--c-ok-soft);
  --color-ok-line: var(--c-ok-line);
  --color-warn: var(--c-warn);
  --color-warn-soft: var(--c-warn-soft);
  --color-warn-line: var(--c-warn-line);
  --color-danger: var(--c-danger);
  --color-danger-soft: var(--c-danger-soft);
  --color-danger-line: var(--c-danger-line);
  --color-muted: var(--c-muted);
}

@layer base {
  :root {
    --c-ok: #3d6b4a;      --c-ok-soft: #f1f6f2;      --c-ok-line: #cfe0d4;
    --c-warn: #8a5a14;    --c-warn-soft: #f8f3ea;    --c-warn-line: #e8dcc2;
    --c-danger: #a23a3a;  --c-danger-soft: #f8eeee;  --c-danger-line: #e9cccc;
    --c-muted: #6b6560;
  }
  .dark {
    --c-ok: #8fc09e;      --c-ok-soft: #17241c;      --c-ok-line: #2a4132;
    --c-warn: #d6b06a;    --c-warn-soft: #251e12;    --c-warn-line: #4a3a1d;
    --c-danger: #e39a9a;  --c-danger-soft: #2a1818;  --c-danger-line: #4d2b2b;
    --c-muted: #a8a29e;
  }
  input[type='checkbox'], input[type='radio'] { accent-color: var(--color-brand-600); }
  .dark input[type='checkbox'], .dark input[type='radio'] { accent-color: var(--color-brand-300); }
}

@custom-variant dark (&:where(.dark, .dark *));

html, body { height: 100%; }

body {
  @apply bg-ink-50 font-sans text-ink-900 antialiased dark:bg-ink-950 dark:text-ink-100;
}

@utility btn {
  @apply inline-flex min-h-10 items-center justify-center gap-2 rounded-lg px-4 text-sm font-medium transition-colors duration-150
    focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600 dark:focus-visible:outline-brand-300
    disabled:cursor-not-allowed disabled:opacity-50 motion-reduce:transition-none;
}

@utility badge {
  @apply inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium;
}

@layer components {
  .card {
    @apply rounded-xl border border-ink-200 bg-white dark:border-ink-800 dark:bg-ink-900;
  }

  /* Botones */
  .btn-primario {
    @apply btn bg-ink-900 text-white hover:bg-ink-700 active:bg-ink-800
      dark:bg-ink-100 dark:text-ink-900 dark:hover:bg-white dark:active:bg-ink-200;
  }
  .btn-secundario {
    @apply btn border border-ink-300 bg-white text-ink-800 hover:bg-ink-50 active:bg-ink-100
      dark:border-ink-700 dark:bg-transparent dark:text-ink-100 dark:hover:bg-ink-800 dark:active:bg-ink-800;
  }
  .btn-fantasma {
    @apply btn text-ink-700 hover:bg-ink-100 active:bg-ink-200 dark:text-ink-200 dark:hover:bg-ink-800 dark:active:bg-ink-700;
  }
  .btn-peligro {
    @apply btn border border-danger bg-transparent text-danger hover:bg-danger-soft;
  }
  .btn-icono {
    @apply btn min-h-10 w-10 px-0;
  }

  /* Formularios */
  .input {
    @apply block min-h-10 w-full rounded-lg border border-ink-300 bg-white px-3 text-sm text-ink-900
      placeholder:text-muted focus:border-brand-600 focus:outline-2 focus:-outline-offset-1 focus:outline-brand-600
      disabled:bg-ink-100 disabled:text-muted aria-invalid:border-danger aria-invalid:outline-danger
      dark:border-ink-700 dark:bg-ink-900 dark:text-ink-100 dark:focus:border-brand-300 dark:focus:outline-brand-300 dark:disabled:bg-ink-800;
  }
  .etiqueta {
    @apply mb-1 block text-sm font-medium text-ink-700 dark:text-ink-300;
  }

  /* Badges: solo lo anómalo lleva color */
  .badge-ok     { @apply badge bg-ok-soft text-ok; }
  .badge-warn   { @apply badge bg-warn-soft text-warn; }
  .badge-danger { @apply badge bg-danger-soft text-danger; }
  .badge-neutro { @apply badge bg-ink-100 text-ink-700 dark:bg-ink-800 dark:text-ink-300; }

  /* Avisos en línea (banners, notas, mensajes dentro de pantallas) */
  .aviso        { @apply rounded-lg border px-3 py-2 text-sm; }
  .aviso-ok     { @apply aviso border-ok-line bg-ok-soft text-ok; }
  .aviso-warn   { @apply aviso border-warn-line bg-warn-soft text-warn; }
  .aviso-danger { @apply aviso border-danger-line bg-danger-soft text-danger; }
  .aviso-info   { @apply aviso border-ink-200 bg-ink-100 text-ink-700 dark:border-ink-800 dark:bg-ink-800 dark:text-ink-200; }

  /* Chips / segmentados (filtros, períodos, orden, tema) */
  .chip {
    @apply inline-flex min-h-9 items-center justify-center gap-1.5 rounded-lg bg-white px-3 text-sm font-medium text-ink-700 ring-1 ring-ink-200
      transition-colors duration-150 hover:bg-ink-50
      focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600
      dark:bg-ink-900 dark:text-ink-200 dark:ring-ink-700 dark:hover:bg-ink-800 dark:focus-visible:outline-brand-300;
  }
  .chip-activo {
    @apply bg-ink-900 text-white ring-ink-900 hover:bg-ink-900 dark:bg-ink-100 dark:text-ink-900 dark:ring-ink-100 dark:hover:bg-ink-100;
  }

  /* Tabla */
  .tabla { @apply w-full text-left text-sm; }
  .tabla th { @apply border-b border-ink-200 px-4 py-3 text-xs font-medium text-muted dark:border-ink-800; }
  .tabla td { @apply border-b border-ink-100 px-4 py-3 dark:border-ink-800; }
  .tabla tbody tr { @apply transition-colors hover:bg-ink-50 dark:hover:bg-ink-800/40; }

  .titulo-pagina { @apply text-xl font-semibold tracking-tight lg:text-2xl; }
}
```

Notas de implementación:
- Los nombres de token existentes (`ink-*`, `brand-*`, `ok`, `warn`, `danger`, `.card`, `.btn-*`, `.badge-*`, `.input`, `.tabla`) se **conservan**: la mayoría de las pantallas cambia de aspecto solo con reemplazar este archivo. Las clases nuevas son `ok-soft/-line`, `warn-*`, `danger-*`, `muted`, `.aviso-*`, `.chip`, `.chip-activo`.
- Si `@apply` de una clase personalizada dentro de otra (`aviso` dentro de `.aviso-ok`) diera error en Tailwind 4, declarar `aviso` con `@utility aviso { ... }` igual que `btn` y `badge`.
- `aria-invalid:outline-danger` sobrescribe el color de foco; verificar que el outline siga visible.

## 7. Metadatos de marca (fuera de styles.css)

| Archivo | Cambio |
|---|---|
| `frontend/src/index.html` | `theme-color` `#db2777` → `#fafaf9` (y agregar `<meta name="theme-color" content="#0c0a09" media="(prefers-color-scheme: dark)">`) |
| `frontend/public/manifest.webmanifest` | `theme_color` → `#fafaf9`, `background_color` → `#fafaf9` |
| `desktop/main.go` línea 95 | `BackgroundColour` → `R: 250, G: 250, B: 249` (hoy 250/250/250, casi igual; solo alinear) |
| `desktop/web/browser/*` | Es salida de build; no editar a mano, se regenera |
| Íconos PWA / `favicon.ico` / `apple-touch-icon` | Hoy probablemente rosa. Rehacer en monocromo (ver pregunta abierta) |
