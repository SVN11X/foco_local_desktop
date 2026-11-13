## [Sin publicar]

- Motor de leases en `internal/limit`: bloqueos temporales que un propietario local de confianza (por ejemplo Guardian) puede pedir. Solo viven en memoria, nunca se guardan en `config.json` y no sobreviven a un reinicio.
- Un lease solo puede cubrir un sitio que el usuario ya tiene como regla activa, no puede ser más amplio que esa regla y se detiene al instante si el usuario la desactiva. Pausar FocoLocal también pausa los leases.
- Cada lease tiene propietario, motivo y expiración de máximo 60 minutos. Es idempotente, admite hasta 20 por propietario y el propietario puede liberarlos todos (liberación de emergencia).
- Los leases no consumen el presupuesto del usuario y no cambian el funcionamiento de sus reglas. Todavía no hay IPC: este cambio no se puede invocar desde fuera de la aplicación.
- Pruebas de expiración, idempotencia, persistencia, cobertura, pausa, concurrencia y presupuesto intacto sin leases.

## [0.3.3] - 2026-11-06

- Nuevos presupuestos independientes «YouTube Shorts» y «YouTube · vídeos»; la detección por URL distingue `/shorts/` de `/watch` sin registrar identificadores de vídeo ni consultas.
- Los accesos de la biblioteca preparan la regla antes de guardarla; los siete sitios de Vídeo y streaming son visibles.
- El tiempo consumido en Shorts no se suma al presupuesto de YouTube normal. Las reglas y consumos anteriores se conservan.
- Pantalla local de bloqueo preparada para la nueva regla con nombre de archivo seguro y título específico.
- Pruebas de normalización, rutas, uso entre navegadores y persistencia de ambos contadores.

## v0.3.2 — Corrección de contraste del menú lateral (Windows)

- El menú lateral mantiene texto de alto contraste en los temas claro, oscuro y sistema. Se separaron los colores semánticos de navegación y tarjetas para evitar que la tinta se convierta en el fondo oscuro al cambiar de tema.
- El elemento seleccionado utiliza azul intenso con texto claro y contraste AA verificado mediante pruebas automatizadas.
- Retiradas las leyendas de navegación «Sin extensiones» y «Sin cortar Internet»; ahora el menú muestra únicamente acciones útiles.
- Se conserva el control de sitios, las reglas, la persistencia y la pantalla de bloqueo sin cambios de lógica.

## [0.3.1] - 2026-11-06

- Biblioteca: 20 pictogramas de marcas reconocibles de Font Awesome Free, locales y con atribución; monogramas para marcas sin icono disponible.
- Experiencia: temas claro, oscuro y sistema (preferencia guardada); encabezado nativo oscuro cuando corresponda.
- Resumen verificable: total de uso realmente contabilizado hoy separado de presupuestos diarios, semanales y mensuales. Registros diarios persistidos.
- Reglas: filas nativas de selección con icono, hoy, presupuesto, estado y progreso.
- Bloqueo: HTML responsivo sin desplazamiento vertical a alturas de ventana comunes y con modo oscuro del sistema.
- Compatibilidad: configuración anterior se conserva. El desglose por día de presupuestos semanales/mensuales anteriores no se puede reconstruir: el primer día se indica la hora de inicio del nuevo contador.
- Calidad: pruebas de contadores, temas y migración; compilación cruzada Windows x64 y análisis estático.

## [0.3.0] - 2026-11-06

### Added
- Nuevo hub de cuatro secciones, tarjetas y botones con iconos de identificación por marca.
- Biblioteca organizada con 36 sitios: seleccionar una sugerencia solo prepara el editor.
- Límites diarios, semanales, mensuales, bloqueo permanente, por horario y por intervalos.
- Presupuestos en minutos u horas, días y selección de meses.
- Seguimiento manual y opcional de una ventana visible en un segundo monitor, con comprobación de actividad y de automatización.
- Pruebas de presupuestos, cronogramas, ciclos y renovación de periodos.

### Fixed
- Separada la activación de la barra de direcciones de la escritura del archivo `file:///` para evitar que Edge busque `ile:///...` en Google.
- Página de bloqueo muestra mensajes según la modalidad (diario/semanal/mensual/horario/intervalo).
- Reinicio de contadores correspondiente al periodo activo de cada regla.
- Mejoras de layout para ajustar la ventana a pantallas más pequeñas.

### Known limitations
- Compilación cruzada y pruebas unitarias realizadas, sin validación visual/funcional en Windows real.
- Sin detección ocular: la segunda pantalla se sigue por elección manual.
- La navegación a un archivo local puede fallar en ciertos navegadores/políticas; se conserva una cubierta visual como alternativa.

## [0.2.3] - 2026-11-06

### Diseño y experiencia
- Panel de escritorio rediseñado con tarjetas claras, navegación lateral, contraste, iconos tipográficos y botones con estados de foco.
- Acceso directo al repositorio SVN11X/foco_local_desktop desde el hub.
- Ocho sitios predefinidos para añadir reglas con un clic, sin reemplazar de forma accidental reglas existentes.
- Totales diarios, estados y lista actualizados cada segundo, sin refrescos de cuatro segundos.
- Página HTML local de bloqueo más acogedora, con recomendaciones de pausas y siguientes actividades, diseño adaptable y sin scripts ni recursos remotos.

### Contabilización y compatibilidad
- Continúa contabilizando solo dominios identificados en la ventana de navegador en primer plano; pestañas ocultas y procesos de fondo no consumen tiempo.
- El refresco visual de un segundo es independiente del sondeo UIA, conservando el coste de lectura previo y sin cambiar los mecanismos de red.
- Se conserva el bloqueo con detención de reproducción y el respaldo visual anteriores.
- Verificación local: pruebas del motor y catálogo, `go vet` Windows y compilación Windows x64. Interfaz nativa pendiente de validación real en Windows.

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
