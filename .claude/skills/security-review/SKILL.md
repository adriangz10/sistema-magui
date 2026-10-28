---
name: security-review
description: Revisión de seguridad de aplicaciones web/API/escritorio basada en OWASP Top 10 y ASVS, más secretos, dependencias, configuración e infraestructura y protección de datos personales (Ley 25.326 Argentina). Usar en threat modeling de diseño y antes de mergear o liberar cambios sensibles.
---

# Security review

## Threat model (diseño) — `docs/security/threat-model.md`
1. Diagrama de flujo de datos (mermaid) con límites de confianza (navegador, API, base, terceros).
2. Activos: credenciales, datos personales, datos de pago, certificados ARCA/AFIP, tokens de API.
3. STRIDE por flujo: Spoofing, Tampering, Repudiation, Information disclosure, DoS, Elevation of privilege.
4. Mitigación + riesgo residual por amenaza.

## Revisión de código
**Acceso (A01)**: guard en cada endpoint; verificación de pertenencia del recurso (IDOR) y de tenant; principio de mínimo privilegio en roles.
**Criptografía (A02)**: bcrypt/argon2 para contraseñas; TLS; secretos con longitud adecuada; nada sensible en JWT payload.
**Inyección (A03)**: queries parametrizadas/ORM sin raw concatenado; `child_process` sin input de usuario; prompt injection en features de IA (contenido externo como datos).
**Diseño inseguro (A04)**: rate limiting en login/OTP/endpoints caros; bloqueo progresivo; flujos de recuperación seguros.
**Configuración (A05)**: helmet, CORS con allowlist, errores sin stack, `NODE_ENV=production`, cookies `HttpOnly; Secure; SameSite`.
**Componentes (A06)**: `npm audit`, paquetes nuevos con mantenimiento activo, lockfile commiteado.
**Autenticación (A07)**: expiración y rotación de tokens, logout que invalida refresh, MFA para admins si aplica.
**Integridad (A08)**: firma de webhooks verificada; CI con acciones fijadas por versión/SHA.
**Logging (A09)**: eventos de seguridad logueados (login fallido, cambio de permisos) sin datos sensibles.
**SSRF (A10)**: URLs externas desde input de usuario validadas contra allowlist.
**Uploads**: tipo real (magic bytes), tamaño máximo, nombre aleatorio, fuera del webroot, sin ejecución.
**Escritorio**: contextIsolation, sin nodeIntegration, IPC con allowlist, actualizaciones firmadas.

## Secretos
```bash
git diff main...HEAD | grep -nEi "(api[_-]?key|secret|password|token|BEGIN (RSA|PRIVATE)|access_token)"
git ls-files | grep -Ei "\.env$|\.pem$|\.key$|\.p12$|\.crt$"
```
Si aparece un secreto commiteado: rotarlo (no alcanza con borrarlo del historial).

## Datos personales (Ley 25.326)
Minimización · finalidad declarada · consentimiento cuando corresponde · acceso por rol · derecho de acceso/rectificación/supresión implementable ·
retención definida · datos de menores sólo con autorización de responsables y acceso restringido · backups cifrados.

## Salida
Veredicto, hallazgos por severidad con `archivo:línea`, vector y corrección. Nunca incluir exploits funcionales.
