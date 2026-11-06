# Diagnóstico del contador en Windows (sin extensiones)

## Qué pasó en la prueba inicial

El hub de FocoLocal estaba visible encima de YouTube. En v0.2.0 solo se cobraba tiempo si el navegador era la ventana en primer plano. Por ello, una reproducción de vídeo que continúa detrás de la aplicación **no equivale a tiempo medido**. La nueva función de monitor flotante permite comprobarlo sin robar el foco.

## Prueba reproducible de dos minutos

1. Cierra la instancia antigua de FocoLocal desde el botón **Salir por completo**. Guarda una copia de `%APPDATA%\FocoLocalDesktop\config.json` si necesitas conservar tus límites.
2. Inicia el ejecutable nuevo y configura `youtube.com` con 2 minutos. Si ya existe la regla, puedes actualizarla sin eliminar datos.
3. Pulsa **Reiniciar contadores de hoy** únicamente si quieres hacer la prueba desde cero.
4. Pulsa **Probar con monitor flotante**. El hub debe ocultarse y quedar el monitor de solo lectura arriba a la derecha.
5. Abre Edge o Chrome y asegúrate de que **YouTube sea la ventana activa**. Mantén el navegador visible, sin clicar en el monitor.
6. El monitor debe indicar `msedge.exe | youtube.com` (o `www.youtube.com`) y el valor `Usados` debe avanzar: `00:05`, `00:10`, etc. Si pone `No se reconoce la barra de direcciones (UIA)`, el navegador está identificado pero aún no se puede leer la URL.
7. Cuando se cumplan dos minutos acumulados, debe aparecer la cubierta de límite. La ventana del navegador conserva sus procesos y tráfico: no se cortan redes ni se cierran sesiones de automatización deliberadamente.
8. Abre el hub desde el icono de FocoLocal en la bandeja y consulta el registro acumulado. El contador se guarda en `%APPDATA%\FocoLocalDesktop\config.json`.

## Cómo interpretar los estados

| Estado en el monitor | Significado |
| --- | --- |
| `youtube.com` y `Usados` avanza | La detección y el motor están trabajando. |
| `Fuera del navegador` | Otra app tiene el foco; **no se cobra tiempo**. |
| `No se reconoce la barra de direcciones (UIA)` | Hay navegador, pero UIA no devolvió el dominio. Reporta navegador, versión, idioma y si es una ventana normal o privada. |
| `Automatización protegida` | El navegador tiene un ancestro de proceso detectado como Python, Node o WebDriver. No se cobra ni bloquea. |
| `SIN REGLA PARA ESTE SITIO` | El dominio activo no coincide con las reglas registradas. |
| `CONTROL PAUSADO` | El control está pausado desde el hub. |

## Límites de diseño

- No hay extensiones ni acceso a los datos internos de reproducción multimedia del navegador. Por ello, la reproducción de vídeo **en segundo plano** no tiene un temporizador propio.
- UI Automation depende del navegador. La compatibilidad con todos los navegadores no se puede prometer sin pruebas por versión.
- En navegadores dirigidos por scripts la exclusión es heurística, no una garantía de aislamiento absoluto. Usa perfiles de automatización independientes para pruebas críticas.
- Si la cubierta desaparece inesperadamente, guarda el estado observado en el monitor y abre una incidencia de compatibilidad con los pasos reproducibles, sin publicar tus URLs privadas.

La prueba debe hacerse con el navegador **en primer plano**. El monitor está diseñado con `WS_EX_NOACTIVATE` para no robarle el foco.

## Prueba de detención de audio (v0.2.2)

1. Cierra cualquier FocoLocal.exe previo mediante **Salir por completo**. Abre la nueva versión.
2. Comprueba que esté activada **Detener página y audio al agotar el tiempo**.
3. Configura `youtube.com` con límite de dos minutos; reinicia contadores solo si deseas empezar la prueba desde cero.
4. En Edge, reproduce un vídeo y deja la pestaña en primer plano. Usa el monitor flotante si deseas verificar el tiempo.
5. Al consumirse el límite, la pestaña debería sustituirse por la página local «Tiempo agotado» y detener el audio. El navegador y las demás pestañas seguirán abiertos.
6. Comprueba que los procesos de Playwright/Python y demás aplicaciones continúen operativos.

**Precaución**: la navegación sustituye el contenido de la pestaña y puede descartar formularios sin guardar. Si necesitas conservarlos, desactiva esta función: el modo visual no detiene el audio. Si Windows bloquea el envío de teclas, la cubierta visual sigue activa como respaldo.

## Verificaciones nuevas (v0.2.3)

1. Comprueba que el panel muestre los tres resúmenes, el botón **↗ GitHub** y ocho sitios predefinidos.
2. Haz clic en **＋ YouTube**: si ya existe la regla, FocoLocal no modifica los minutos anteriores y muestra cómo editarlos. Si no existe, la agrega con su límite sugerido.
3. Configura YouTube en 2 minutos y pulsa **Monitor flotante**. Activa Edge en YouTube: los segundos deberían progresar aproximadamente `00:01`, `00:02`, `00:03` cada segundo.
4. Cambia a otra aplicación y espera 15 segundos: el contador **no** debe subir. El vídeo puede seguir reproduciéndose, pero no se cobra tiempo mientras Edge no está en primer plano.
5. Regresa a Edge, termina el límite y confirma la nueva página local con **¿Qué puedes hacer ahora?** y sin audio de la pestaña sustituida.
6. Desde el hub, abre **↗ GitHub** para verificar que lleva al repositorio oficial en el navegador que Windows tenga configurado.

El contador mide el **uso de un sitio en primer plano**, no atención ocular ni actividad constante de ratón/teclado. Ver un vídeo en una pestaña visible y activa cuenta; ver ese mismo vídeo con otra app en primer plano no cuenta.
