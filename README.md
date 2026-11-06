# FocoLocal Desktop

**Control de tiempo de navegación en Windows desde una sola aplicación, sin extensiones y sin suscripciones.**

![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white) ![Platform](https://img.shields.io/badge/Windows-10%20%2F%2011-0078D4?logo=windows&logoColor=white) ![License](https://img.shields.io/badge/license-MIT-555555) ![Status](https://img.shields.io/badge/estado-evaluaci%C3%B3n-orange)

> **Estado: v0.2.1 de evaluación.** El motor pasó pruebas unitarias y el ejecutable se puede compilar, pero la integración UI Automation/Win32 **no se ha verificado en Windows real**. No se puede garantizar compatibilidad con todos los navegadores ni aislamiento absoluto de automatizaciones.

## Qué hace

- Administra en un **hub local** reglas por dominio (`youtube.com → 30 min/día`).
- Comparte el tiempo acumulado cuando se reconoce el dominio activo en navegadores distintos.
- Al agotarse el límite, cubre visualmente **esa ventana** del navegador; no corta Internet ni cierra procesos.
- Funciona sin extensiones, cuentas de usuario, suscripción, proxy, cambios DNS, archivo HOSTS ni reglas del firewall.
- Conserva reglas y contadores en el propio PC, sin telemetría del proyecto.
- Ofrece exclusiones de PID y detección **heurística** de navegadores iniciados por automatizaciones.
- Incluye un **monitor flotante no activable** para comprobar qué dominio se detecta y ver el tiempo consumido mientras el navegador tiene el foco.

## Instalación y uso

### Para usuarios

1. Obtén `FocoLocal.exe` de una **Release verificada** del repositorio cuando exista; si estás evaluando el prototipo, compílalo con las instrucciones de abajo.
2. Guárdalo en una ubicación permanente, por ejemplo `C:\Apps\FocoLocal\FocoLocal.exe`.
3. Ejecútalo en Windows 10/11 con tu usuario normal: **sin extensiones ni permisos de administrador**.
4. Agrega un dominio y minutos: `youtube.com`, `30` y **Guardar límite**.
5. Si quieres, activa **Iniciar con Windows** desde el hub.
6. Para verificar un límite sin perder el foco del navegador, pulsa **Probar con monitor flotante**: el hub se oculta y queda una pequeña ventana de lectura en pantalla. Activa YouTube y observa el contador. El hub se abre desde su icono en la bandeja del sistema.
7. Para salir definitivamente, usa **Salir por completo** desde la bandeja del sistema.

Consulta [la prueba guiada de dos minutos](docs/GUIA_PRUEBA_WINDOWS.md) si el contador permanece a cero.

**Alcance:** el contador mide **tiempo de navegador en primer plano**. Si el hub de FocoLocal está por delante de YouTube, **no descontará el tiempo** aunque el vídeo continúe reproduciéndose. Tampoco contabiliza descargas, API, pestañas ocultas ni actividad en segundo plano. Detectar reproducción de vídeo en segundo plano en todos los navegadores sin extensiones no está implementado.

### Para desarrolladores

Requisitos: Go 1.23+ para compilar y Windows para validar la interfaz.

```powershell
git clone <URL-DE-TU-REPOSITORIO> FocoLocalDesktop
cd FocoLocalDesktop
go test ./...
go vet ./...
$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
go build -trimpath -ldflags='-s -w -H windowsgui' -o FocoLocal.exe ./cmd/focolocal
```

El módulo se denomina `focolocal-desktop` de forma local para no inventar una URL GitHub que todavía no está definida. Al publicar, actualiza el módulo y el import interno si quieres utilizar el proyecto como dependencia externa. El `.exe` se ignora en Git y se distribuye solo como artefacto/release.

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

### ¿Qué sucede cuando se agota el tiempo? (v0.2.2)

**Modo predeterminado: detener página y audio.** Cuando se supera un límite en una pestaña **activa**, FocoLocal la redirige a una pantalla HTML local de «Tiempo agotado». Esto detiene el audio de esa pestaña sin cerrar el navegador ni afectar la red, otras aplicaciones o navegadores automatizados que se reconozcan como tales. No instala extensiones.

Este modo **puede descartar formularios y cambios sin guardar** en la pestaña restringida. Para evitarlo, desmarca **«Detener página y audio al agotar el tiempo»** en el hub: mantendrás la cubierta visual anterior, pero los vídeos podrían continuar reproduciéndose por detrás. El programa no detecta de forma universal audio de pestañas en segundo plano.

Por seguridad, la sustitución requiere dos detecciones consecutivas de la misma página y una ventana de navegador que siga en primer plano. Si falla la inyección de teclado, se utiliza la cubierta visual como respaldo. Nunca se usa un firewall, proxy, DNS ni cierre de procesos.
