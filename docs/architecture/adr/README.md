# Registro de decisiones de arquitectura (ADR)

Plantilla y reglas: skill `architecture-decision`. Un ADR aceptado no se edita: se reemplaza por uno nuevo.
Las decisiones anteriores a este registro (monolito modular hexagonal en Go, MySQL + SQLite, escritorio offline con
cola de ventas) están en `docs/PLAN.md` y `docs/CONVENCIONES.md`.

| N.º | Título | Estado | Historias |
|---|---|---|---|
| [0001](0001-orquestar-edicion-de-compra-desde-compras.md) | Orquestar la edición de una compra desde compras, en una transacción y por puertos | Propuesto | US-001 |
| [0002](0002-concurrencia-edicion-compra-version-y-bloqueo-de-lotes.md) | Versión optimista en la compra y bloqueo pesimista de lotes | Propuesto | US-001a/b/c |
| [0003](0003-tipo-de-movimiento-edicion-compra.md) | Tipo de movimiento `EDICION_COMPRA` | Propuesto | US-001b/c |
| [0004](0004-baja-logica-de-renglones-de-compra.md) | Baja lógica del renglón (`compra_item.quitado_en`) | Propuesto | US-001c |
| [0005](0005-auditoria-de-ediciones-de-compra-en-json.md) | Auditoría `compra_edicion` con JSON de cambios | Propuesto | US-001a/b/c |
