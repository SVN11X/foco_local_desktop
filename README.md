# FocoLocal Desktop

**Control de tiempo de navegación en Windows desde una sola aplicación, sin extensiones y sin suscripciones.**

![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white) ![Platform](https://img.shields.io/badge/Windows-10%20%2F%2011-0078D4?logo=windows&logoColor=white) ![License](https://img.shields.io/badge/license-MIT-555555) ![Status](https://img.shields.io/badge/estado-evaluaci%C3%B3n-orange)

> **Estado: v0.3.3 (YouTube Shorts con presupuesto independiente y detección por ruta; interfaz nativa pendiente de validación visual en Windows).** La detección de YouTube y el bloqueo por sustitución de pestaña fueron probados con versiones anteriores. Esta actualización requiere comprobar la interfaz, la navegación local y el modo de dos pantallas en un Windows real. No garantiza compatibilidad con todos los navegadores ni detección de mirada.

## Funcionalidades de v0.3.3

- **YouTube Shorts separado de YouTube normal:** dos reglas independientes, `youtube.com/shorts` y `youtube.com`. Los Shorts abiertos en `/shorts/ID` (incluidos `www.` y `m.`) se contabilizan solo en Shorts; los vídeos `/watch?v=ID` y las demás páginas de YouTube se contabilizan solo en YouTube normal. **No se suman ambas reglas simultáneamente.**
- En **Explorar sitios → Vídeo y streaming**, selecciona «YouTube Shorts» o «YouTube · vídeos» para configurar sus límites sin activar automáticamente la regla. Se conserva el presupuesto existente de `youtube.com` (o `www.youtube.com`).
- **Limitación:** si un Short se reproduce en `/watch?v=ID`, la barra de direcciones no revela que ese vídeo es corto y se contabiliza como vídeo normal. Sin extensiones o integración con YouTube no es posible identificar universalmente el tipo de vídeo.

## Funcionalidades anteriores

- **Una sola aplicación de escritorio**: cuatro vistas (Resumen, Explorar sitios, Crear regla, Preferencias), con navegación e iconos de color propios de cada sitio; sin extensiones, cuentas, servicios de red ni dependencias adicionales.
- **Biblioteca de 37 opciones** agrupados en vídeo y streaming, redes sociales, comunidades, juegos y ocio, compras, y noticias y trabajo. **Seleccionar NO guarda ni activa la regla**: abre el editor con valores sugeridos.
- **Modalidades:** minutos y horas, presupuesto diario, semanal y mensual; bloqueo todo el día; franjas horarias con días laborables/fines de semana; intervalos de uso/descanso; restricción de meses (`1,2,12` o `4-9`). Se aplican únicamente tras pulsar **Guardar regla**.
- **Uso compartido entre navegadores** cuando sus barras de direcciones se pueden leer mediante UI Automation. El contador de la interfaz se actualiza cada segundo.
- **Dos pantallas:** por defecto **solo cuenta el navegador en primer plano**, esté en el monitor que esté. Puedes seleccionar **Seguir otra pantalla** para dar prioridad a esa ventana visible en otro monitor mientras haya actividad reciente en el PC (aunque el navegador principal tenga foco). **No detecta la mirada** y no pretende medir atención ocular. Para detenerlo, pulsa el mismo botón. Esta selección no se guarda entre reinicios.
- **Bloqueo de página sin audio:** separa el atajo para enfocar la barra de direcciones y la escritura del archivo local para evitar perder la letra inicial de `file:///...`. Si Windows impide la navegación, mantiene el bloqueo superpuesto como respaldo. La página local muestra alternativas y el tipo de renovación de la regla.
- **Protección de automatizaciones:** no modifica DNS, firewall, hosts ni proxy; mantiene exclusiones por PID y detección heurística de scripts. La protección no es infalible si un script comparte el navegador interactivo.
- **Persistencia:** reglas y contadores en `%APPDATA%\FocoLocalDesktop\config.json`, **independientes de la carpeta y nombre del EXE**. Puedes reemplazar el `.exe` cuando actualices; no elimines la carpeta de configuración si quieres conservar tus datos.

## Tipos de reglas

| Opción | Comportamiento |
| --- | --- |
| Diario | Presupuesto por día; renovación a medianoche local |
| Semanal | Presupuesto compartido de lunes a domingo; renovación el lunes |
| Mensual | Presupuesto de todo el mes; renovación el día 1 |
| Todo el día (bloquear) | Bloquea siempre mientras esté habilitada la regla |
| Por horario | Permite el sitio solo en la franja `HH:MM–HH:MM` y días elegidos, además del presupuesto diario |
| Por intervalos | Presupuesto diario con pausas obligatorias tras determinados minutos de uso; la ventana puede necesitar abrirse de nuevo después |
| Meses | Se puede restringir en qué meses está disponible una regla; el campo vacío implica todos |

El tiempo de vídeo **en una pestaña que no está siendo usada** no se carga. En modo de segunda pantalla se utiliza la selección manual de ventana más un umbral de actividad del equipo; esto es una aproximación y no confirma cuál pantalla estás mirando.

## Instalación y uso

### Para usuarios

1. Obtén `FocoLocal.exe` de una **Release verificada** del repositorio cuando exista; si estás evaluando el prototipo, compílalo con las instrucciones de abajo.
2. Guárdalo en una ubicación permanente, por ejemplo `C:\Apps\FocoLocal\FocoLocal.exe`.
3. Ejecútalo en Windows 10/11 con tu usuario normal: **sin extensiones ni permisos de administrador**.
4. En **Explorar sitios** selecciona una web, o entra a **Crear regla** e introduce el dominio manualmente.
5. Escoge si el límite será diario, semanal, mensual, por horario, por intervalos o bloqueo permanente; configura el presupuesto en minutos u horas, y pulsa **Guardar regla**.
6. En **Resumen**, revisa el contador cada segundo y edita o elimina una regla cuando quieras. El **monitor flotante** permite observar el tiempo sin quitar el foco al navegador.
7. En **Preferencias**, configura la reproducción, el inicio con Windows y el seguimiento opcional de una ventana en la segunda pantalla. Para salir usa **Salir por completo**.

Consulta [la prueba guiada de dos minutos](docs/GUIA_PRUEBA_WINDOWS.md) si el contador permanece a cero.

**Alcance:** el contador mide por defecto **tiempo de navegador en primer plano**. En modo de seguimiento manual de ventana secundaria puede contabilizar un navegador visible aunque otra aplicación tenga el foco; deja de hacerlo si el PC permanece inactivo durante unos 3 minutos. No mide realmente tu mirada, la actividad de API ni la reproducción en pestañas ocultas.**

### Para desarrolladores

Requisitos: Go 1.23+ para compilar y Windows para validar la interfaz.

```powershell
git clone https://github.com/SVN11X/foco_local_desktop.git FocoLocalDesktop
cd FocoLocalDesktop
go test ./...
go vet ./...
$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
go build -trimpath -ldflags='-s -w -H windowsgui' -o FocoLocal.exe ./cmd/focolocal
```

El módulo local se llama `focolocal-desktop`; el repositorio oficial está en [SVN11X/foco_local_desktop](https://github.com/SVN11X/foco_local_desktop). No es necesario cambiar el módulo para compilar el ejecutable. El `.exe` se ignora en Git y se distribuye solo como artefacto/release.

## Compatibilidad real

El programa intenta leer el dominio desde la barra de direcciones mediante **Windows UI Automation (UIA)**.

| Medio | Alcance actual |
| --- | --- |
| Chrome, Edge, Firefox, Brave, Opera, Vivaldi y derivados | Detección por proceso; lectura de URL **condicionada** a controles UIA accesibles |
| Modo incógnito/privado | No garantizado: depende del navegador y su UIA |
| Pestañas en segundo plano / navegadores integrados | No medidas de forma general |
| Playwright, Selenium, Python, Node.js y tareas de red | No se bloquean conexiones, pero una ventana automatizada reutilizada podría recibir la cubierta |
| Otros navegadores / navegadores sin controles accesibles | Sin soporte garantizado; no se cobra tiempo si el dominio no puede leerse |

**No es un control parental a prueba de evasión.** Puede cerrarse desde el área de notificación. La detección por UIA y las exclusiones son best-effort; verifica con [la matriz manual de pruebas](docs/TESTING.md).

## Seguridad y privacidad

Las reglas y los contadores se guardan en `%APPDATA%\FocoLocalDesktop\config.json`. No se guarda el historial completo ni se inspecciona HTTPS; la aplicación procesa el dominio visible solo para aplicar límites. Consulta [PRIVACY](docs/PRIVACY.md) y [SECURITY](SECURITY.md). Para eliminar los datos, cierra la aplicación y elimina la carpeta local indicada.

## Proyecto

| Recurso | Contenido |
| --- | --- |
| [Arquitectura](docs/ARCHITECTURE.md) | Componentes, flujo y límites de la solución |
| [Pruebas](docs/TESTING.md) | CI y escenarios Windows por navegador |
| [Hoja de ruta](docs/ROADMAP.md) | Trabajo pendiente, sin promesas de compatibilidad |
| [Contribuir](CONTRIBUTING.md) | Configuración, PR y criterios de calidad |
| [Soporte](SUPPORT.md) | Cómo solicitar ayuda e informar fallos |
| [Seguridad](SECURITY.md) | Reportes privados de vulnerabilidades |
| [Código de conducta](CODE_OF_CONDUCT.md) | Normas de convivencia |
| [Publicar versiones](docs/RELEASING.md) | Flujo manual de pre-releases verificadas |
| [Preparar GitHub](docs/GITHUB_SETUP.md) | Ajustes que el repositorio no puede activar solo |
| [Historial](CHANGELOG.md) | Cambios por versión |

## Tecnologías y referencias

Implementación propia en **Go (biblioteca estándar), Win32 y Microsoft UI Automation**. Referencias oficiales: [Microsoft UI Automation](https://learn.microsoft.com/windows/win32/winauto/uiauto-uiautomationoverview), [Win32 CreateWindowEx](https://learn.microsoft.com/windows/win32/api/winuser/nf-winuser-createwindowexw), [Go y compilación Windows](https://go.dev/wiki/WindowsCrossCompiling), [buenas prácticas GitHub](https://docs.github.com/en/repositories/creating-and-managing-repositories/best-practices-for-repositories).

Software bajo [licencia MIT](LICENSE). No afiliado a Microsoft, Google, Mozilla o GitHub.

### ¿Qué sucede cuando se agota el tiempo? (v0.2.3)

**Modo predeterminado: detener página y audio.** Cuando se supera un límite en una pestaña **activa**, FocoLocal la redirige a una pantalla HTML local de «Tiempo agotado». Esto detiene el audio de esa pestaña sin cerrar el navegador ni afectar la red, otras aplicaciones o navegadores automatizados que se reconozcan como tales. No instala extensiones.

Este modo **puede descartar formularios y cambios sin guardar** en la pestaña restringida. Para evitarlo, desmarca **«Detener página y audio al agotar el tiempo»** en el hub: mantendrás la cubierta visual anterior, pero los vídeos podrían continuar reproduciéndose por detrás. El programa no detecta de forma universal audio de pestañas en segundo plano.

Por seguridad, la sustitución requiere dos detecciones consecutivas de la misma página y una ventana de navegador que siga en primer plano. Si falla la inyección de teclado, se utiliza la cubierta visual como respaldo. Nunca se usa un firewall, proxy, DNS ni cierre de procesos.

### Diseño UX y contabilización (v0.2.3)

- Interfaz Win32 renovada con navegación lateral, tarjetas de resumen, iconografía textual accesible, mayor jerarquía tipográfica, acciones etiquetadas y retroalimentación tras cambios. Sin WebView, bibliotecas externas, extensiones o solicitudes de red.
- Ocho accesos rápidos incluyen límites sugeridos; la entrada manual sigue disponible.
- La lista y los indicadores se actualizan cada **1 segundo** (el análisis del navegador se ejecuta por separado). El cambio visual no debe interpretarse como precisión cronométrica absoluta: `WM_TIMER` de Windows es aproximado.
- El tiempo se descuenta **únicamente con una URL reconocida de un navegador en primer plano**, nunca porque haya audio en segundo plano. Un vídeo visible en primer plano sí cuenta aunque no pulses el teclado o el ratón continuamente.
- La nueva página HTML local muestra alternativas para continuar el día, mantiene la navegación sin scripts y no recopila información adicional.
- El prototipo usa una ventana compacta para evitar ocultar controles en pantallas habituales; el escalado DPI y la accesibilidad de diferentes temas deben verificarse en Windows real.


## v0.3.2: legibilidad en temas oscuro y claro

Se corrigió un fallo de contraste en el menú lateral: la tinta del menú ya no comparte el color de las tarjetas del área de contenido. Navegación, textos secundarios y botones de acción mantienen alto contraste en Claro, Oscuro y Sistema; el botón seleccionado usa un azul oscuro estable. Se quitaron del menú dos leyendas técnicas redundantes. No se modificaron el motor de contabilidad, las reglas, el esquema de configuración ni la navegación de bloqueo.

## v0.3.1: métricas y apariencia

- **Uso real registrado hoy** se almacena en `daily_usage_ms`, separado de los presupuestos semanales y mensuales. Al actualizar desde versiones antiguas se recupera el día de reglas diarias; el desglose histórico de reglas mensuales/semanales no puede reconstruirse con precisión.
- En Preferencias se puede elegir Sistema, Claro u Oscuro; la opción Sistema lee la preferencia de Windows.
- Los logotipos reconocibles de 20 plataformas se incluyen sin conexiones externas. Otras plataformas mantienen un monograma identificativo; la biblioteca no implica relación ni patrocinio de esas marcas. Ver `THIRD_PARTY_NOTICES.md`.
- La página de descanso usa CSS adaptable por altura y anchura; se mantiene local y sin scripts.
