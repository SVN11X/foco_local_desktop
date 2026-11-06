# Publicación de versiones

**No publicar como estable** mientras la interfaz Win32 y UI Automation no estén verificadas en Windows real. Hasta entonces usar la etiqueta *pre-release* de GitHub.

1. Revisa los cambios mediante PR y comprueba CI en verde.
2. Ejecuta [pruebas manuales](TESTING.md) con al menos Chrome, Edge y una automatización de Playwright en Windows.
3. Actualiza `CHANGELOG.md`, `README.md` y la versión de `CITATION.cff`.
4. Crea una etiqueta semántica (p. ej. `v0.2.1`) **solo cuando el código y documentación coincidan**.
5. Compila desde el commit etiquetado. No reutilices binarios de origen incierto.

```powershell
$env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
go test ./...
go build -trimpath -ldflags='-s -w -H windowsgui' -o FocoLocal.exe ./cmd/focolocal
Get-FileHash -Algorithm SHA256 .\FocoLocal.exe
```

6. Crea manualmente una **GitHub Release (pre-release)** con notas de limitaciones, `FocoLocal.exe` y suma SHA-256 de ese mismo archivo.
7. Compara la suma del archivo publicado con el hash del build. Nunca subas `.exe` directamente al historial Git.

La CI actual publica **artefactos temporales de Actions, no GitHub Releases**: evita distribuir automáticamente software de escritorio aún no validado.
