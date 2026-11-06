# Arquitectura

## Objetivos no negociables

1. **Un único hub de escritorio**, sin extensiones.
2. **Cuota compartida** entre navegadores reconocidos cuando una página permanece activa en primer plano.
3. **Sin bloqueos de red**: los scripts y tareas en segundo plano deben seguir trabajando.
4. **Privacidad local**: guardar solo reglas y agregados, no el historial completo.

## Componentes

```text
cmd/focolocal/main_windows.go
    └── internal/windows
         ├── app_windows.go       interfaz, timers y cubierta visual
         ├── uia_windows.go       dominio a partir de UI Automation
         ├── process_windows.go   heurística de navegadores automatizados
         └── winapi_windows.go    Win32 y controles del proceso
             └── internal/limit/engine.go
                  ├── reglas por dominio
                  ├── cuotas compartidas por día
                  └── persistencia JSON local
```

Solo el motor de límites es portable. La interfaz depende de Windows y sus archivos usan el sufijo `_windows.go`.

## Medición

- Se identifica la ventana **en primer plano** y, cuando es posible, su dominio.
- La regla más específica tiene prioridad y suma tiempo en una única cuota diaria.
- Un cambio de ventana o un intervalo de suspensión prolongado no debe añadir tiempo retroactivo.
- La cubierta visual bloquea una ventana identificada; no filtra el tráfico HTTPS ni el uso en pestañas ocultas.

## Seguridad del desarrollo

- Evitar código con privilegios elevados; preferir APIs de usuario y HKCU.
- No registrar URLs completas ni información sensible por defecto.
- No bloquear DNS, hosts, cortafuegos, proxies ni procesos externos.
- Tratar UI Automation como best-effort: diferentes navegadores/idiomas pueden cambiar controles.
- Probar explícitamente Playwright, Selenium, Python y VS Code en cada cambio relevante.

## Restricciones actuales

La detección de navegadores mediante UI Automation no está verificada en todas las versiones. Las exclusiones de automatización por ascendencia del proceso son heurísticas. No es control parental invulnerable ni un bloqueador universal de sitios.

Consulta [decisión 0001](decisions/0001-sin-bloqueo-de-red.md) y [matriz de pruebas](TESTING.md).
