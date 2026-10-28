---
name: client-handover
description: Entrega y cierre de un proyecto o fase con el cliente: acta de entrega, transferencia de accesos y credenciales, capacitación, garantía, propuesta de mantenimiento y retrospectiva interna. Usar al terminar un MVP, una fase contratada o el proyecto completo (Gate G5).
---

# Entrega al cliente y cierre

## 1. Verificación previa
- [ ] Todo el alcance contratado (`docs/proposal/propuesta.md`) entregado o con desvío acordado por escrito
- [ ] Último release en producción y estable ≥ 48 h
- [ ] Manuales de usuario y admin al día
- [ ] Backups funcionando y restauración probada
- [ ] Sin bugs críticos/altos abiertos

## 2. Paquete de entrega
- Acceso al repositorio (o código fuente según contrato) y documentación (`docs/`).
- Inventario de accesos: dominio, DNS, hosting, base de datos, registry, cuentas de terceros (ARCA/AFIP, MercadoPago, Meta/WhatsApp, Google), con dueño de cada cuenta.
  **Las credenciales se transfieren por un canal seguro (gestor de contraseñas compartido), nunca por email/chat ni dentro de documentos.**
- Lista de vencimientos: dominio, certificados, tokens, suscripciones.

## 3. Capacitación
Agenda por rol (usuarios / administradores), duración, material (manual + video corto por tarea), ejercicio práctico con datos de prueba.

## 4. Acta de entrega — `docs/proposal/acta-entrega-<fase>.md`
```markdown
# Acta de entrega — <Proyecto> · <Fase>
Fecha · Cliente · Responsable
## Alcance entregado (vs. propuesta)
## Pendientes acordados (con fecha)
## Accesos transferidos (listado, sin contraseñas)
## Capacitación realizada
## Garantía: <plazo> desde la fecha, cubre <defectos del alcance>, no cubre <cambios/nuevas funciones>
## Conformidad del cliente: ______________________
```

## 5. Continuidad
Propuesta de abono de mantenimiento (horas, SLA de respuesta por severidad, hosting, actualizaciones de seguridad) y backlog de fase 2 priorizado.

## 6. Retro interna — `docs/plan/retro-cierre.md`
Estimado vs. real por módulo, qué riesgos se materializaron, qué reutilizar (código, plantillas), qué ajustar en la fábrica (agentes/skills).
