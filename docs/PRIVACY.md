# Privacidad y datos

FocoLocal Desktop está diseñado para funcionar localmente. No requiere cuentas, extensión, suscripción ni servidor remoto.

## Información procesada

- Dominio extraído de la barra de direcciones de una ventana activa cuando UI Automation lo permite.
- Regla de tiempo configurada por el usuario.
- Tiempo agregado de uso por día/dominio.
- Preferencias de interfaz y exclusiones de PID para la sesión.

## Información que no se pretende recolectar

- Contenido HTML de la página, formularios o contraseñas.
- URLs completas e historial cronológico de navegación.
- Tráfico HTTPS, cookies o paquetes de red.
- Telemetría hacia servidores del proyecto.

## Almacenamiento y eliminación

Los contadores y reglas se guardan en `%APPDATA%\FocoLocalDesktop\config.json` y permanecen en el PC. Para eliminar datos, cierra el programa por completo y borra esa carpeta. Los contadores antiguos se depuran de acuerdo con la lógica del motor; consulta el código fuente para el comportamiento exacto.

**Alcance:** descripción del código v0.2.3, no una auditoría independiente. Las versiones modificadas o distribuidas por terceros pueden comportarse de forma diferente.

## Páginas locales de bloqueo (v0.2.2)

Cuando está activado «Detener página y audio», FocoLocal genera una página HTML local
`bloqueado-<dominio>.html` junto a `config.json` en el directorio de configuración
del usuario. No tiene JavaScript, recursos de Internet ni telemetría. La página
contiene únicamente el dominio restringido, no la URL completa, el historial
ni el contenido del sitio. La navegación a este archivo sustituye la pestaña
activa (puede descartar datos sin guardar) y no bloquea conexiones de otros
procesos.

En v0.2.3 los sitios predefinidos son una lista estática local y el enlace GitHub solo abre el navegador cuando lo pulsas. No hay consultas automáticas al repositorio.
