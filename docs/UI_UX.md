# Interfaz FocoLocal v0.2.3

El objetivo es una interfaz nativa y comprensible que funcione sin extensiones, sin cuentas y sin instalar componentes adicionales. La identidad visual utiliza tarjetas claras, navegación lateral, jerarquía tipográfica y símbolos del sistema, siguiendo las recomendaciones generales de Microsoft para aplicaciones Windows.

## Decisiones de experiencia de usuario

- **Pocos pasos:** agregar dominios manualmente o con uno de ocho accesos directos. Se confirma antes de borrar; los accesos directos existentes no sobrescriben límites.
- **Estado visible:** tres tarjetas muestran cantidad de sitios, tiempo agregado del día y estado general. Lista e indicadores se actualizan cada segundo.
- **Semántica comprensible:** siempre aparece el dominio y el tipo de acción. El color del estado está acompañado de texto.
- **Navegación:** campos de edición, botones y casillas conservan el manejo de teclado de Win32. En los botones personalizados el foco se dibuja explícitamente.
- **Privacidad:** la interfaz no utiliza recursos web, iconos descargados, telemetría, frameworks embebidos ni APIs de cuentas.
- **Rendimiento:** `WM_TIMER` refresca UI a 1 Hz; el scanner conserva su ritmo separado de 1,3 segundos. Las reglas se evalúan solo con muestra vigente en la ventana de navegador activa.
- **Bloqueo amable:** archivo HTML local, adaptable a pantalla pequeña, con alternativas de uso del tiempo y sin JavaScript ni trackers.
- **Transparencia:** el límite de primer plano no demuestra actividad humana real; un vídeo en pantalla puede estar contando aunque no se pulse una tecla. El tiempo en segundo plano no se cuenta.

## Validación pendiente en Windows

La compilación cruzada a Windows x64 no prueba la reproducción visual ni el comportamiento de la UI en todas las escalas. Se recomienda revisar en Windows 10 y Windows 11: escala al 100 % y al 125 %, resolución 1366×768 y superior; foco de teclado de los botones; casillas de verificación en sidebar; contraste en temas de alto contraste; apertura del enlace GitHub; y sustitución de pestaña cuando el contador llega a cero.

## Referencias oficiales

- [Windows apps best practices](https://learn.microsoft.com/en-us/windows/apps/get-started/best-practices)
- [Windows design guidelines](https://learn.microsoft.com/en-us/windows/apps/design/)
- [Win32 timers](https://learn.microsoft.com/en-us/windows/win32/winmsg/timers)
