---
name: ux-flows
description: Diseñar flujos de usuario, wireframes de pantallas con todos sus estados, design tokens y criterios de accesibilidad a partir de historias de usuario. Usar en la fase de diseño, antes de implementar UI, dejando el resultado en docs/ux/.
---

# Flujos y pantallas

## 1. Flujos (`docs/ux/flujos.md`)
Un diagrama por tarea principal:

```mermaid
flowchart TD
  A[Inicio: lista de ventas] --> B{¿Cliente existe?}
  B -- Sí --> C[Seleccionar cliente]
  B -- No --> D[Alta rápida de cliente]
  C --> E[Cargar productos] --> F[Confirmar] --> G[Ticket / Factura]
```

## 2. Pantalla (`docs/ux/pantallas/<nombre>.md`)

```markdown
# Pantalla: <nombre>  (US-NNN)
Objetivo del usuario:
Ruta: /ventas/nueva   · Roles: vendedor, admin

## Wireframe
+--------------------------------------------------+
| ← Volver        Nueva venta            [Guardar] |
| Cliente: [ buscar…            ▾]  [+ Nuevo]      |
| Productos                                         |
| | Producto        | Cant | Precio   | Subtotal | |
| Total: $ 0,00                                     |
+--------------------------------------------------+

## Campos
| Campo | Tipo | Obligatorio | Validación | Mensaje de error |

## Estados
- Vacío: …   - Cargando: skeleton   - Error de red: …   - Sin permisos: …
- Éxito: …   - Offline (si aplica): …

## Acciones e interacciones
## Accesibilidad
## Responsive (mobile / desktop)
```

## 3. Design tokens (`docs/ux/design-tokens.md`)
Colores (primario, secundario, neutros, éxito, advertencia, error) con contraste AA verificado,
tipografía (familia, escala), espaciado (escala 4/8), radios, sombras, breakpoints.

## Checklist
- [ ] Cada historia con UI tiene pantalla y estados.
- [ ] Formatos locales: fechas dd/mm/aaaa, moneda `$ 1.234,56`, CUIT/DNI con máscara.
- [ ] Acciones destructivas con confirmación y opción de deshacer cuando sea posible.
- [ ] Contraste ≥ 4.5:1, foco visible, todo operable por teclado, targets táctiles ≥ 44px.
