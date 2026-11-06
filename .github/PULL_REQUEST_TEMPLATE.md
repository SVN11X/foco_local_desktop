## Resumen
<!-- ¿Qué cambia y por qué? -->

## Tipo de cambio
- [ ] Corrección de error
- [ ] Nueva función
- [ ] Documentación / mantenimiento
- [ ] Seguridad / privacidad

## Validación
- [ ] `gofmt -l cmd internal` no informa archivos
- [ ] `go test ./...` pasa
- [ ] `go vet ./...` pasa
- [ ] Compilé `GOOS=windows GOARCH=amd64` cuando corresponde
- [ ] Probé en Windows real si modifiqué UI Automation, Win32 o el bloqueo visual
- [ ] Comprobé que Playwright/Python y otros procesos automáticos no se interrumpen

## Evidencia, riesgos y compatibilidad
<!-- Describe escenarios probados y limitaciones. No adjuntes datos personales. -->

## Checklist
- [ ] No incorporé extensiones, bloqueos DNS/hosts, firewall ni proxy sin discutir el cambio de arquitectura
- [ ] No agregué binarios, secretos, registros ni configuración privada al repositorio
- [ ] Actualicé README / CHANGELOG / docs si aplica
- [ ] Revisé el impacto de permisos y privacidad
