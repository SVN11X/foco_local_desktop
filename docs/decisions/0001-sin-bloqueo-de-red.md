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
