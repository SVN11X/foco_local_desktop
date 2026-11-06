# ADR-0001: no bloquear conexiones de red

- **Estado:** aceptada para v0.2.x
- **Fecha:** 2026-11-06

## Contexto

Una regla basada en DNS, proxy, HOSTS o firewall afectaría también a Python, Playwright, Selenium y descargas automáticas del mismo dominio. El requisito principal es limitar **navegación manual** sin interrumpir automatizaciones.

## Decisión

FocoLocal usa lectura de dominio visible por UI Automation y una cubierta de ventana. No intercepta paquetes de red ni mata procesos externos.

## Consecuencias

**Ventajas:** sin extensiones, sin privilegios elevados ni cambios a la red; las tareas en segundo plano siguen funcionando.

**Limitaciones:** navegadores inaccesibles para UIA quedan fuera; el bloqueo no es infalible, no limita pestañas en segundo plano y una ventana manual controlada por automatización puede verse afectada por la cubierta. Se requiere prueba por navegador.

## Ampliación v0.2.2: detener contenido al agotar tiempo

La cubierta visual no detenía vídeos ni sonido. Se incorpora un modo **optable desde el hub**, activado de manera predeterminada, que reemplaza *solo la pestaña interactiva y en primer plano* por una página `file:` local usando el teclado estándar de Windows. No necesita extensión ni interrumpe la red. La medida puede descartar cambios sin guardar; por ello existe la opción de volver al bloqueo visual. Solo se intenta con detección reciente y estable del dominio y respetando las exclusiones de automatizaciones.
