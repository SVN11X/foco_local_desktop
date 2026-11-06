## [0.2.2] - 2026-11-06

### Correcciones y seguridad
- Nuevo modo de bloqueo **Detener página y audio** (predeterminado): sustituye únicamente la pestaña activa agotada por un archivo HTML local, sin extensiones, cambios DNS ni cierre de procesos.
- Mantiene el modo anterior **Solo bloqueo visual** como alternativa para páginas con formularios o trabajo sin guardar.
- Acciones de teclado solo si el navegador sigue en primer plano, la identificación es reciente y dos lecturas UIA consecutivas coinciden; las automatizaciones reconocidas siguen excluidas.
- Nueva pantalla local de bloqueo sin scripts, recursos remotos ni solicitudes de red.
- El bloqueo visual sigue siendo el respaldo si Windows impide enviar eventos de teclado o el navegador no es compatible.

### Limitaciones
- Reemplazar una pestaña puede descartar datos o cambios sin guardar en esa página.
- Las pestañas que ya reproducían audio en segundo plano no se detectan universalmente sin extensiones; un bloqueo de la pestaña activa no garantiza silenciar otras pestañas.
- El envío de teclas puede fallar por UIPI si el navegador se ejecuta con privilegios distintos o si el usuario cambia de ventana.

## [0.2.1] - 2026-11-06

### Corregido
- El hub ahora explica cuándo tiene el foco y por qué el contador no avanza.
- Se diferencia el último dominio leído correctamente del navegador observado sin URL.

### Agregado
- Monitor flotante no activable para observar la URL detectada y el consumo acumulado en segundos mientras el navegador conserva el foco.
- Guía de diagnóstico reproducible en Windows, sin extensiones ni Go local.

### Límites conocidos
- No se mide reproducción de vídeo en pestañas o ventanas en segundo plano.
- La lectura de dominios depende de la accesibilidad UI Automation que exponga cada navegador y requiere pruebas en Windows real.

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
