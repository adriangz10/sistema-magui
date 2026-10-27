# Rediseño visual minimalista — Sistema Magui

Pedido: "estilos más minimalistas, sin tanto color llamativo, sencillo, lindo de ver, colores más neutros".
Alcance: **solo visual**. No cambia funcionalidad, rutas, textos ni estructura de navegación.
Tokens y CSS listos para pegar: ver `docs/ux/design-tokens.md`.

## 1. Inventario del estado actual

Fuente: revisión de `styles.css` y de todos los componentes en `frontend/src/app/**` (búsqueda de clases de color, gradientes y sombras).

| Aspecto | Hoy | Problema |
|---|---|---|
| Color de marca | `brand` rosa fucsia (#db2777) en: botón primario, logo, ítem de nav activo, foco, enlaces, paginador activo, pestaña activa, barras de gráfico, íconos de estado vacío, theme-color, manifest | Es lo primero que se ve y está en todas partes |
| Neutros | `ink` = slate azulado (#f8fafc ... #020617) | Frío, algo "corporativo" |
| Semánticos | Mezcla de tokens (`text-ok`, `text-warn`, `text-danger`) y paleta por defecto de Tailwind (`green-50/100/300/400/800/950`, `amber-*`, `red-*`) | Verdes/ámbares/rojos saturados; cada uso repite el par `dark:` a mano; inconsistente |
| Gradiente | Login: `bg-gradient-to-br from-brand-50 via-white to-ink-100` (y versión oscura) | Decorativo |
| Sombras | `shadow-sm` (card, botón primario), `shadow-xs` (input), `shadow` y `shadow-lg` (logo en login), `shadow-lg` (toast, tooltip), `shadow-xl` (dropdown buscador), `shadow-2xl` (modal, menú "Más") | Demasiado peso visual |
| Radios | `rounded-2xl` en cards, `rounded-xl` botones/inputs, `rounded-3xl` en bottom-sheet, `rounded-full` en badges | Muy redondeado, estilo "app de juguete" |
| Otros | `backdrop-blur` en header y barra de cobro móvil; encabezados de tabla en MAYÚSCULAS con tracking; `font-bold` en KPIs y logo; ring de foco con color de marca claro (`ring-brand-200`) | Ruido |
| Contraste | `text-ink-500` sin variante `dark:` en ~50 lugares (≈3.2:1 en oscuro); `text-ink-400` usado como texto | Incumple AA |
| Badges | `badge-ok` verde para "Confirmada" y "Vigente" (lo normal) | Pinta lo que no necesita atención |

Colores de Tailwind por defecto aún en uso (deben desaparecer): `green-*`, `amber-*`, `red-*`.
Archivos que los usan: `layout/shell.component.ts`, `shared/ui/avisos.component.ts`, `features/inicio/inicio.page.ts`, `features/estadisticas/estadisticas.page.ts`, `features/vender/vender.page.ts`, `features/vender/panel-cobro.component.ts`, `features/ventas/ventas.page.ts`, `features/inventario/inventario.page.ts`, `features/configuracion/configuracion.page.ts`, `features/productos/producto-form.component.ts`, `features/categorias/empresas.component.ts`, `features/caja/caja.page.ts`, `shared/ui/buscador-producto.component.ts`, `shared/ui/campo.component.ts`, `shared/graficos/tarjeta-dato.component.ts`, `features/ingresar/ingresar.page.ts`.

## 2. Propuesta (resumen)

- Neutros cálidos (stone) + **un** acento azul-verdoso apagado, usado solo en foco, enlaces, seleccionado y gráficos.
- Botón principal casi negro (invertido en oscuro). Secundario con borde fino. Fantasma sin borde. Peligro: contorno rojo apagado (no relleno).
- Cards planas: borde fino, sin sombra, `rounded-xl`.
- Color solo para información: error, sin stock, vencido, anulada (rojo apagado); stock bajo, por vencer, offline, pendiente (ámbar apagado); confirmación puntual (verde apagado).
- Lo "normal" va en neutro: Confirmada, Vigente, En línea.
- Tabla más calma: encabezados en minúscula/normal, sin mayúsculas.
- Oscuro con la misma lógica: superficies `ink-900` sobre fondo `ink-950`, semánticos que cambian solos vía variables CSS.

## 3. Mapeo "clase actual → clase nueva"

### 3.1 Reemplazos globales (buscar y reemplazar, revisar caso por caso)

| Actual | Nuevo | Nota |
|---|---|---|
| `text-ink-500`, `text-ink-400` (cuando es texto) | `text-muted` | Elimina cualquier `dark:text-ink-*` acompañante. Si es ícono decorativo puede quedar `text-ink-400` |
| `text-ok dark:text-green-300/400` | `text-ok` | Quitar el par `dark:` |
| `text-warn dark:text-amber-300` | `text-warn` | idem |
| `text-danger dark:text-red-300/400` | `text-danger` | idem |
| `bg-green-50 ... text-ok ... dark:bg-green-950 dark:text-green-300` | `aviso-ok` | Más `flex`/`items-center`/`justify-between` si los tenía |
| `bg-amber-50 ... text-warn ... border-amber-200 dark:...` | `aviso-warn` | idem |
| `bg-red-50 ... text-danger ... border-red-200 dark:...` | `aviso-danger` | idem |
| `bg-brand-50 text-brand-700 dark:bg-brand-950 dark:text-brand-200` (nav activo) | `bg-ink-100 text-ink-900 dark:bg-ink-800 dark:text-ink-50` | El nav activo pasa a neutro |
| `text-brand-600 dark:text-brand-300` (nav inferior activo) | `text-ink-900 dark:text-ink-50` | Agregar `font-semibold` al texto activo |
| `text-brand-700 hover:underline dark:text-brand-300` (enlaces) | `text-brand-600 hover:underline dark:text-brand-300` | Enlaces conservan el acento |
| `bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900` / `bg-white text-ink-700 ring-1 ring-ink-200 dark:...` (chips) | `chip` + `[class.chip-activo]="activo"` | Se repite en estadísticas, inventario, configuración (tema), orden |
| `bg-brand-600 text-white` (paginador activo) | `bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900` | |
| `border-brand-600 text-brand-700 dark:text-brand-300` (pestaña activa) | `border-ink-900 text-ink-900 dark:border-ink-100 dark:text-ink-50` | Pestañas neutras |
| `bg-brand-50 ... dark:bg-brand-950` + `border-brand-500` (opción seleccionada) | sin cambio de clases | El token nuevo ya lo vuelve sutil (azul-gris muy claro) |
| `accent-brand-600` | quitar (lo hace `styles.css`) o dejar | Es opcional |
| `shadow-sm`, `shadow-xs`, `shadow` | eliminar | |
| `shadow-lg`, `shadow-xl`, `shadow-2xl` | `shadow-md` | Solo en flotantes |
| `rounded-2xl` (cards, avisos, bloques de total) | `rounded-xl` | Íconos cuadrados grandes → `rounded-xl` |
| `rounded-3xl` | `rounded-2xl` (solo bottom-sheet, `rounded-t-2xl`) | |
| `rounded-xl` en botones/inputs/chips | `rounded-lg` | Ya lo resuelven las clases base |
| `font-bold` | `font-semibold` | |
| `backdrop-blur`, `bg-white/85`, `bg-ink-900/85`, `bg-white/95`, `bg-ink-900/95` | `bg-white dark:bg-ink-900` | Fondos sólidos |
| `bg-ink-950/50` (overlay del menú "Más") | `bg-ink-950/40` | |

### 3.2 Componentes base

| Componente | Claro | Oscuro | Clase |
|---|---|---|---|
| Botón primario | fondo `ink-900`, texto blanco | fondo `ink-100`, texto `ink-900` | `.btn-primario` |
| Botón secundario | blanco, borde `ink-300`, texto `ink-800` | transparente, borde `ink-700` | `.btn-secundario` |
| Botón fantasma | sin fondo, hover `ink-100` | hover `ink-800` | `.btn-fantasma` |
| Botón peligro | contorno y texto `danger`, hover `danger-soft` | idem | `.btn-peligro` |
| Input | blanco, borde `ink-300`, `rounded-lg`, sin sombra; foco: borde y outline `brand-600` | fondo `ink-900`, borde `ink-700`, foco `brand-300` | `.input` |
| Card | blanco, borde `ink-200`, `rounded-xl`, sin sombra | `ink-900`, borde `ink-800` | `.card` |
| Tabla | th: `text-xs font-medium text-muted`; filas con línea `ink-100`; hover `ink-50` | líneas `ink-800` | `.tabla` |
| Badge | pastilla `rounded-md` suave. Solo `warn`/`danger` coloreados; resto `neutro` | variables | `.badge-*` |
| Nav lateral | fondo blanco, borde derecho `ink-200`; ítem activo `bg-ink-100 text-ink-900 font-medium`; ítems `text-ink-600`, hover `bg-ink-50` | `ink-900`, activo `ink-800` | en `shell.component.ts` |
| Toast | `rounded-xl border bg-white shadow-md`; borde izquierdo no; tipo se indica con ícono coloreado + texto `ink-900` | `ink-900` | `avisos.component.ts` |
| Modal | `rounded-xl bg-white shadow-md`, backdrop `ink-950/40`, cabecera y pie separados por `border-ink-100` | `ink-900` | `modal.component.ts` |
| KPI | card plana; etiqueta `text-sm text-muted`; valor `text-2xl font-semibold tabular-nums text-ink-900`; tendencia en `ok`/`danger` con ícono flecha | variables | `tarjeta-dato.component.ts` |
| Carrito/cobro | card plana; total `text-3xl font-semibold`; falta en `danger`; vuelto en `aviso-ok`; botón Cobrar `btn-primario h-12` | | `panel-cobro`, `lista-carrito`, `vender.page` |

### 3.3 Toasts (`avisos.component.ts`, objeto `clases`)

```ts
exito: 'border-ink-200 bg-white text-ink-900 dark:border-ink-700 dark:bg-ink-900 dark:text-ink-100',
error: 'border-danger-line bg-danger-soft text-ink-900 dark:text-ink-100',
info:  'border-ink-200 bg-white text-ink-900 dark:border-ink-700 dark:bg-ink-900 dark:text-ink-100',
```
Y colorear solo el ícono con una clase por tipo: éxito `text-ok`, error `text-danger`, info `text-muted`. Texto siempre `ink-900`/`ink-100` para mantener contraste. Reemplazar `shadow-lg` por `shadow-md`.

## 4. Pautas por pantalla / archivo

Orden recomendado: 1) `styles.css` + metadatos, 2) `shared/ui`, 3) `layout`, 4) pantallas, 5) gráficos. Con solo el paso 1 ya cambia ~70 % del aspecto.

### Base
- **`src/styles.css`**: reemplazar por el bloque de `design-tokens.md` §6.
- **`src/index.html`, `public/manifest.webmanifest`**: `theme-color`/`theme_color`/`background_color` según tokens §7.
- **`desktop/main.go`**: `BackgroundColour` 250/250/249 (opcional). No tocar `desktop/web/browser`.

### shared/ui
- **`avisos.component.ts`**: objeto `clases` y sombra según §3.3. Íconos con color por tipo. El texto del confirmar usa `btn-peligro` (contorno) si `peligro`.
- **`modal.component.ts`**: `rounded-2xl` → `rounded-xl`; `shadow-2xl` → `shadow-md`; mantener `border-ink-100` en cabecera y pie. Backdrop del `<dialog>`: `::backdrop` con `bg-ink-950/40`. Botón cerrar: `btn-icono btn-fantasma`.
- **`motivo-modal.component.ts`**: `text-ink-600`/`text-ink-300` → `text-muted`.
- **`campo.component.ts`**: `text-ink-400/500` → `text-muted`; error: `text-danger` (sin `dark:text-red-400`) con ícono de alerta delante.
- **`input-dinero.component.ts`**: el prefijo "$" `text-ink-400` → `text-muted`.
- **`buscador-producto.component.ts`**: dropdown `rounded-xl shadow-xl` → `rounded-xl shadow-md`; ítem resaltado `bg-brand-50 dark:bg-brand-950` → `bg-ink-100 dark:bg-ink-800`; `text-red-400` fuera; `text-ink-400/500` → `text-muted`.
- **`paginador.component.ts`**: página activa `bg-brand-600 text-white` → `bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900`; resto de grises sin cambio de estructura; `text-ink-500/400` → `text-muted`; agregar `aria-current="page"` si falta.
- **`tabla.component.ts`**: `bg-ink-100/800` del esqueleto de carga → `bg-ink-100 dark:bg-ink-800` con `animate-pulse`; textos secundarios `text-muted`. Estado vacío: ícono `text-ink-400` en círculo `bg-ink-100`, sin color de marca.
- **`encabezado.component.ts`**: subtítulo `text-muted`; título con `.titulo-pagina`.

### layout
- **`shell.component.ts`**
  - Skip-link: `bg-brand-600` → `bg-ink-900 text-white`.
  - Logo "M" (sidebar l.27 y header móvil l.54): `bg-brand-600 text-white shadow` → `bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900`, sin sombra, `rounded-lg`, `font-semibold`.
  - Sidebar: `rounded-xl` de ítems → `rounded-lg`; activo según §3.1.
  - Header (l.52): `bg-white/85 backdrop-blur` → `bg-white dark:bg-ink-900`.
  - Chip de conexión (l.66): en línea = neutro (`bg-ink-100 text-ink-700 dark:bg-ink-800 dark:text-ink-300`, punto `bg-ink-400`); sin conexión = `bg-warn-soft text-warn`. Mantener el texto.
  - Banners (l.78, 87, 92): `aviso-warn` / `aviso-danger` pero a todo el ancho: conservar `border-b`, `rounded-none`, `px-4 lg:px-8` (agregar `rounded-none`).
  - Barra inferior (l.103-106): activo según §3.1; el resto `text-muted`.
  - Menú "Más" (l.120-138): overlay `bg-ink-950/40`; hoja `rounded-t-2xl shadow-md`; ítems `rounded-xl` → `rounded-lg`; "Salir" `text-danger hover:bg-danger-soft` (sin `dark:hover:bg-red-950`).

### features
- **`ingresar.page.ts`** (login): quitar gradiente → `bg-ink-50 dark:bg-ink-950`; logo `size-12 rounded-xl bg-ink-900 text-white text-xl font-semibold` sin `shadow-lg`; card con `.card p-6 sm:p-8`; error con `aviso-danger` y ícono; botón `btn-primario w-full h-11`.
- **`inicio.page.ts`**: alerta de stock (l.47-52) → `card` con borde `border-warn-line bg-warn-soft` (o `aviso-warn flex items-center gap-3 p-4`), ícono y enlace `text-warn`; quitar `hover:border-amber-300` → `hover:border-ink-300`. Enlace "Ver estadísticas completas" `text-brand-600 dark:text-brand-300`. Accesos rápidos/KPIs: cards planas con ícono `text-ink-500` dentro de círculo `bg-ink-100 dark:bg-ink-800` (sin colores por tarjeta). Textos `text-ink-500` → `text-muted`.
- **`vender/vender.page.ts`**: barra fija móvil (l.44) `bg-white dark:bg-ink-900` sólido; "falta/listo" (l.48) `text-danger` / `text-muted` (cuando está listo no hace falta verde); pantalla de éxito (l.64-75): ícono `text-ok` sin `dark:`, bloque del vuelto → `aviso-ok rounded-xl p-4`, nota offline `text-warn`.
- **`vender/panel-cobro.component.ts`**: card plana; método de pago (si usa botones/radios) como `chip`; "falta" `text-danger`; vuelto `aviso-ok flex justify-between`; `problema` (l.78) → `aviso-warn`; botón `btn-primario h-12 w-full text-base`; total `text-3xl font-semibold tabular-nums`; separadores `border-ink-100 dark:border-ink-800`.
- **`vender/lista-carrito.component.ts`**: estado vacío (l.14) círculo `bg-ink-100 text-ink-500 dark:bg-ink-800 dark:text-ink-400` (no brand); badge "Stock registrado" queda `badge-warn`; botón quitar `text-muted hover:text-danger`.
- **`ventas/ventas.page.ts`**: badge estado: `Confirmada` → `badge-neutro`, `Anulada` → `badge-danger` (l.76 y l.92); aviso de anulación (l.96) → `aviso-danger`; renglón de ganancia (l.130) `text-ok` sin `dark:` (o mejor neutro con `font-medium`); nota (l.134) `text-warn`.
- **`inventario/inventario.page.ts`**: tarjetas de alerta (l.29-43): cards planas; el **número** solo se colorea si es > 0 (`text-danger` / `text-warn`, ya está así, quitar `dark:`); `hover:border-red-300`/`amber-300` → `hover:border-ink-300`; `font-bold` → `font-semibold`; pestañas (l.52) según §3.1; filtros (l.63) → `chip`; stock del movimiento (l.122) `text-ink-900` para ingreso y `text-danger` para egreso (el verde se elimina; el signo +/− ya informa).
- **`inventario/estados.ts`**: `VIGENTE: 'badge-ok'` → `'badge-neutro'` (lo normal no se pinta). Actualizar `estados.spec.ts` si aserta la clase.
- **`inventario/ajuste-stock.component.ts`**: opción seleccionada (`bg-brand-50 border-brand-500`) queda, ya sutil; `bg-ink-50 / dark:bg-ink-800/60` de resumen OK; textos `text-muted`.
- **`productos/productos.page.ts`**, **`producto-form.component.ts`**: `text-ok`/`text-green-400` (l. del margen) → `text-ok` solo; errores `text-danger`; resumen `bg-ink-50` OK; `divide-ink-100` OK.
- **`productos/historial-precios.component.ts`**: `text-ink-500` → `text-muted`.
- **`categorias/categorias.page.ts`**, **`empresas.component.ts`**, **`ajuste-precios.component.ts`**: solo reemplazos §3.1 (muted, danger sin `dark:`, selección con tokens nuevos).
- **`compras/compras.page.ts`**, **`compra-form.component.ts`**: reemplazos §3.1; sin color propio.
- **`caja/caja.page.ts`**: diferencia/faltante `text-danger`; sobrante sin color especial (texto normal, con signo); checkbox queda.
- **`estadisticas/estadisticas.page.ts`**: selector de período (l.37) y orden (l.106) → `chip`; nota ámbar (l.65) → `aviso-warn`; `text-white` en activos ya cubierto por `.chip-activo`; títulos de sección `text-base font-semibold`; KPIs vía `tarjeta-dato`.
- **`configuracion/configuracion.page.ts`**: selector de tema (l.79) → `chip`; "✓ Ya la estás usando..." `text-ok` sin `dark:`; estado del servidor: "En línea" neutro, "Con problemas" `text-danger`; `text-ink-500` → `text-muted`.

### shared/graficos
- **`tarjeta-dato.component.ts`**: card plana; tendencia `text-ok`/`text-danger` sin `dark:` y con flecha (▲/▼ o ícono) para no depender del color.
- **`barras-horizontales.component.ts`**: barra `bg-brand-600 dark:bg-brand-500` → `bg-brand-600 dark:bg-brand-400`; pista `bg-ink-100 dark:bg-ink-800`; alto `h-2` OK; etiquetas `text-muted`.
- **`grafico-columnas.component.ts`**: columnas `fill-brand-600 dark:fill-brand-400` (con `opacity-40` para las no activas); ejes/rejilla `stroke-ink-200 dark:stroke-ink-800` (rejilla más tenue que el eje `stroke-ink-300`); textos de ejes `fill-ink-500 dark:fill-ink-400` (cumple AA sobre ink-900 solo con 400 en oscuro); tooltip `shadow-lg` → `shadow-md`.

## 5. Criterios de aceptación visual / QA

1. `grep -rE "(green|amber|red|rose|pink)-[0-9]" frontend/src/app` sin resultados (excepto specs).
2. `grep -rE "gradient|shadow-(lg|xl|2xl)|backdrop-blur|font-bold|rounded-3xl"` sin resultados.
3. `grep "dark:text-(green|amber|red)"` sin resultados.
4. Ningún relleno de color de marca en botones, logo ni nav.
5. Axe sin violaciones de contraste en claro y oscuro en: ingresar, inicio, vender (carrito + éxito), ventas (detalle anulada), inventario (las 3 pestañas), estadísticas, caja, configuración.
6. Foco por teclado visible en cada control (Tab recorriendo nav, filtros, tabla, modal).
7. Probar offline (chip de conexión y banner) y modal de confirmación peligrosa en ambos temas.
8. Revisión en 360 px de ancho y en la app de escritorio (Wails) con el mismo build.
9. Sin cambios en tests funcionales; solo actualizar `estados.spec.ts` si aserta clases.

## 6. Riesgos

- Botón principal oscuro puede parecer "pesado" a quien esperaba color: se valida con una captura de `vender` y `inicio` antes de generalizar.
- Cambio de `VIGENTE` y `Confirmada` a neutro: confirmar que el negocio no usa el verde como señal visual habitual.
- Íconos de la app (PWA, favicon, instalador del escritorio) siguen rosados hasta rehacerlos.
