# FocoLocal Desktop

## Unreleased · preparación del repositorio GitHub
- Archivos de comunidad: contribuciones, conducta, seguridad, soporte, accesibilidad y citación.
- Formularios Issue, plantilla de PR, Dependabot y CI con pruebas, vet, formato y compilación.
- CI limitada a permisos de lectura, acciones fijadas por commit y artefactos temporales con SHA-256.
- Documentación de privacidad, arquitectura, pruebas, publicación y ajustes de GitHub.
- Se reemplaza el módulo `example.com` por el módulo local `focolocal-desktop`, sin inventar un repositorio remoto.
- No hay cambios previstos en la lógica del limitador ni nuevos métodos de bloqueo.

## v0.2.0 (2026-11-06)
- Aplicación de escritorio nativa Windows x64, sin extensiones y sin dependencias de runtime.
- Gestión local de límites diarios por dominio y subdominio con presupuesto compartido entre navegadores.
- Lectura del control de dirección activo mediante Microsoft UI Automation.
- Cubierta visual de bloqueo; botón para salir hacia otra pestaña.
- Protección heurística de navegadores lanzados desde Python/Node/WebDriver y exclusión manual por PID.
- Minimización a bandeja, inicio automático opcional, persistencia local, limitación a una instancia.
- Compilación cruzada PE32+ verificada; pruebas unitarias del motor.
- Pendiente: prueba manual de integración UIA / GUI en Windows, soporte de UIA para cada navegador y versión.
