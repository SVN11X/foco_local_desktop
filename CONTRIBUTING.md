# Contribuir a FocoLocal Desktop

¡Gracias por colaborar! El objetivo es ofrecer un limitador gratuito para Windows, sin extensiones y sin interferir con la programación ni con automatizaciones.

## Antes de comenzar

1. Lee [README](README.md), [arquitectura](docs/ARCHITECTURE.md), [pruebas](docs/TESTING.md) y [seguridad](SECURITY.md).
2. Para una vulnerabilidad, utiliza el canal privado de `SECURITY.md`, **no** una incidencia pública.
3. Abre una incidencia para acordar cambios grandes (método de detección, bloqueo o almacenamiento).

## Preparar el entorno

- Go 1.23 o superior para desarrollar.
- Windows 10/11 para validar UI Automation/Win32; en Linux puedes ejecutar las pruebas del motor y hacer compilación cruzada.
- No hace falta instalar dependencias de terceros: el módulo actual usa solo la biblioteca estándar.

```powershell
go test ./...
go vet ./...
gofmt -w ./cmd ./internal
$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
go build -trimpath -ldflags='-s -w -H windowsgui' -o FocoLocal.exe ./cmd/focolocal
```

## Enviar cambios

- Crea una rama descriptiva: `fix/uia-firefox`, `docs/setup`, `feat/reports`.
- Mantén los PR pequeños, incluye pruebas para la lógica del motor y explica cómo reproducir resultados.
- Verifica manualmente en Windows cualquier modificación de UIA, superposición de ventanas, autoarranque o detección de procesos.
- No subas ejecutables, capturas con URLs privadas, cookies, tokens, historiales de navegación ni archivos de `%APPDATA%`.
- El autor conserva sus derechos de autor y acepta que su contribución se distribuya bajo MIT.

## Decisiones de arquitectura

**No** utilizar extensiones de navegador, bloqueo global de dominios, manipulación del DNS o firewall ni cierre forzoso de procesos sin una discusión explícita. Esas decisiones protegen las automatizaciones del usuario. Consulta [ADR-0001](docs/decisions/0001-sin-bloqueo-de-red.md).

## Normas de convivencia

Consulta el [código de conducta](CODE_OF_CONDUCT.md). Los reportes de compatibilidad deben incluir navegador y versión, pero no información personal de navegación.
