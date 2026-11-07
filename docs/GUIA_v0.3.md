# Guía de prueba — FocoLocal 0.3.0

## Antes de empezar

1. Cierra FocoLocal anterior desde **Salir por completo**. Comprueba que no quede un proceso en ejecución.
2. Guarda una copia de `%APPDATA%\FocoLocalDesktop\config.json` si quieres conservar un respaldo; no elimines la carpeta.
3. Sustituye el ejecutable por el `FocoLocal.exe` del ZIP. Si aparece SmartScreen, comprueba procedencia y hash antes de decidir si lo ejecutas.

## Pruebas de aceptación

- **Vista:** navega por Resumen, Explorar sitios, Crear regla y Preferencias. Comprueba que los controles no se corten, especialmente con escala de Windows 125 %.
- **Biblioteca:** en Explorar sitios, cambia de categoría y elige YouTube. Verifica que **no se active ninguna regla aún**. El dominio aparecerá en Crear regla; el guardado solo ocurrirá al hacer clic en Guardar regla.
- **Regla básica:** configura 2 minutos diarios. Abre YouTube en Edge, mantén esa ventana en primer plano. El contador debería aumentar cada segundo. Cambia a otra aplicación: debería detenerse.
- **Tiempo:** crea una regla de 1 hora, una semanal y otra mensual. Verifica su etiqueta y que no se confundan los periodos.
- **Horario:** permite de 09:00 a 18:00. Fuera de esas horas FocoLocal sustituirá la pestaña si está activa; prueba un dominio de prueba para evitar perder contenido no guardado.
- **Intervalos:** 1 minuto de uso y 2 de pausa, presupuesto 10 minutos. Tras llegar al intervalo, la pestaña de prueba debe detenerse y permitir su navegación después de la pausa.
- **Dos pantallas:** activa un navegador de la segunda pantalla, vuelve a FocoLocal y pulsa Seguir otra pantalla. Con otro programa activo, verifica que el contador avanza si el navegador fijado es visible y el PC tiene actividad reciente. Pulsa de nuevo para detenerlo.
- **Bloqueo:** revisa que la pestaña local comience por `file:///` (no por `google.com/search?q=ile...`) y que se detenga el audio de esa pestaña. El encabezado y consejos deben estar visibles.
- **Automatizaciones:** comprueba que Python, VS Code y Playwright continúen trabajando fuera del navegador personal.

## Reportar errores

Incluye versión, navegador, dos monitores (sí/no), escalado DPI, tipo de regla y captura. Evita incluir rutas privadas, dominios sensibles o procesos de trabajo internos.


## YouTube: vídeos y Shorts por separado (v0.3.3)

1. En **Explorar sitios → Vídeo y streaming**, selecciona **YouTube Shorts**.
2. Ajusta su presupuesto (por ejemplo, 10 minutos diarios) y pulsa **Guardar regla**.
3. Repite para **YouTube · vídeos** con otro presupuesto (por ejemplo, 60 minutos).
4. Abre `https://www.youtube.com/shorts/ID` en el navegador activo para consumir solo Shorts.
5. Al visitar `https://www.youtube.com/watch?v=ID`, consume solo la regla normal.

Las reglas anteriores de YouTube no se eliminan ni se restablecen.
**Advertencia:** si una página Shorts se abre en el reproductor normal `/watch`,
se contará como vídeo normal. La detección depende de que el navegador muestre
la ruta completa mediante Windows UI Automation. Las pestañas en segundo plano
no se contabilizan.
