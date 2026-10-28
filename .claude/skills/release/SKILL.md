---
name: release
description: Preparar y liberar una versión: cálculo SemVer desde los commits, CHANGELOG, notas de versión para el cliente, checklist de salida a producción, tag, plan de rollback y verificación post-deploy. Usar cuando un conjunto de historias está listo para producción.
---

# Release

## 1. Versión
```bash
git describe --tags --abbrev=0          # último tag
git log --oneline <ultimo-tag>..HEAD    # cambios
```
`BREAKING CHANGE` → major · `feat` → minor · sólo `fix`/`perf` → patch. Antes de 1.0.0, breaking → minor.

## 2. CHANGELOG.md (Keep a Changelog)
```markdown
## [1.4.0] - 2026-11-03
### Agregado
- Emisión de factura B desde la venta (US-021)
### Cambiado
### Corregido
- Redondeo del total con descuentos (BUG-014)
### Seguridad
```

## 3. Notas de versión — `docs/releases/vX.Y.Z.md`
```markdown
# Versión X.Y.Z — <fecha>
## Para el cliente
Qué hay de nuevo (en lenguaje simple) · Qué se corrigió · Qué tienen que hacer ustedes (si algo)
## Técnico
Migraciones · Variables de entorno nuevas · Cambios de infraestructura · Dependencias actualizadas
## Rollback
Pasos concretos y consideraciones sobre migraciones
```

## 4. Checklist de salida (Gate G4)
- [ ] PRs aprobados por code-reviewer y security-engineer
- [ ] OK de QA con reporte en `docs/qa/`
- [ ] Historias aceptadas por PO
- [ ] CI verde en `main`
- [ ] Migraciones probadas en staging (incluido `down` o plan alternativo)
- [ ] Backup de producción reciente verificado
- [ ] Variables de entorno de prod cargadas
- [ ] Ventana de despliegue acordada con el cliente (evitar horarios pico / cierre de caja)
- [ ] Aprobación humana explícita

## 5. Liberar (sólo con G4 aprobado)
```bash
git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z
```

## 6. Post-deploy
Smoke test del camino crítico, `/health`, revisar logs y errores 30 min, confirmar integraciones (emitir comprobante de prueba si aplica), avisar al cliente con las notas.
