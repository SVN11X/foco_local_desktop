# Cómo interpretar los contadores

## Total registrado hoy

`USO REAL REGISTRADO HOY` no suma las cifras que aparecen en los presupuestos de las reglas. Es la suma de `daily_usage_ms[YYYY-MM-DD]` para cada dominio contabilizado por el motor ese día.

- Sólo hay una muestra facturable por instante; se selecciona la regla más específica coincidente.
- Un navegador sin foco no se cobra, excepto una ventana **elegida manualmente** mediante «Seguir otra pantalla» que continúe visible y con actividad reciente.
- Vídeo reproduciéndose detrás del hub no genera tiempo de uso.
- Páginas abiertas mediante automatizaciones detectadas están excluidas si «Proteger scripts» está activo.
- Al agotar el presupuesto deja de descontarse tiempo; el contenido de esa pestaña se sustituye por la página local cuando la configuración lo permite.
- Los sitios quitados durante el día pueden conservar su uso histórico contabilizado, por lo que el total de hoy puede incluirlos.

## Fila de una regla

- `Hoy`: sólo el tiempo efectivamente acumulado este día.
- `Período`: acumulado contra el presupuesto de la regla. En reglas diarias suele coincidir con Hoy; en reglas semanales/mensuales puede ser mayor.
- La barra refleja **período / presupuesto**. Una regla de bloqueo permanente no tiene presupuesto temporal.

## Migración desde v0.3.0

El formato anterior sólo guardaba acumulados por período. El desglose histórico diario de reglas **semanales/mensuales** no puede reconstruirse con precisión; el primer día de la actualización se indica «Desde la actualización: HH:MM» si existían consumos previos de tales reglas. Desde ese momento los registros diarios se guardan separadamente en `daily_usage_ms` dentro de `%APPDATA%\FocoLocalDesktop\config.json` y se mantienen al reemplazar el EXE.

## Límites

`DailyUsage` registra actividad del navegador, no atención visual ni tiempo de reproducción en otra ventana. El modo de segunda pantalla es una excepción manual y no un detector de mirada. Ningún bloqueo se realiza mediante DNS, proxy o firewall.
