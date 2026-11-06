# Política de seguridad

## Versiones con mantenimiento

| Versión | Estado |
| --- | --- |
| `0.2.x` | Versión de evaluación; se aceptan informes de seguridad |
| `< 0.2` | Sin mantenimiento comprometido |

**No se garantiza soporte de seguridad con plazos de respuesta determinados.** El proyecto aún requiere validación de integración en Windows.

## Reportar una vulnerabilidad

1. Si el repositorio tiene habilitado **Private vulnerability reporting**, abre `Security → Advisories → Report a vulnerability` en GitHub.
2. Si no está habilitado, contacta al mantenedor por un canal **privado** indicado en su perfil y solicita habilitar el reporte privado.
3. No publiques exploits, información sensible, URL visitadas, tokens, configuraciones privadas ni PoC que permitan abuso en Issues.

Incluye impacto, versión, pasos de reproducción con datos ficticios y una propuesta de mitigación si existe. No realices pruebas sobre equipos o cuentas ajenas.

## Superficie de riesgo

- Lectura de controles de accesibilidad del navegador mediante Windows UI Automation.
- Ventanas superpuestas del bloqueo visual.
- Persistencia local en `%APPDATA%\FocoLocalDesktop`.
- Modificación opcional del inicio automático por usuario en Windows.
- Distinción heurística de navegadores de automatización: **no garantiza aislamiento completo**.

No se usa proxy, DNS, archivo hosts ni firewall. No hay telemetría ni envío deliberado de historial a servidores; revisa [Privacidad](docs/PRIVACY.md).

## Reglas para colaboradores

No incluyas credenciales en commits ni PR. Para dependencias futuras, comprueba licencias, vulnerabilidades y actualización de versiones. Las modificaciones sensibles deben pasar por revisión y pruebas reales en Windows.
