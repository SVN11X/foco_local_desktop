# Checklist al publicar en GitHub

La estructura de archivos ya está preparada. Estas opciones requieren configuración **en la interfaz del repositorio remoto** y no pueden activarse con solo añadir archivos.

1. Crea el repositorio vacío (sin README/licencia inicial si vas a subir esta carpeta). Reemplaza `module focolocal-desktop` por el módulo GitHub real si quieres que sea importable por terceros; cambia también los imports internos.
2. Completa `About`: descripción clara, `Topics` (por ejemplo `windows`, `go`, `productivity`, `open-source`, `desktop-app`), y enlace de descarga **solo si existe una Release verificada**.
3. En `Settings → General`, habilita **Issues**; **Discussions** es opcional según haya moderación.
4. Protege la rama principal con **Rulesets/Branch protection**: PR requerido, al menos una revisión cuando haya colaboradores, impedir force-push y exigir los checks de CI, después de verificar sus nombres reales.
5. En `Settings → Actions → General`, restringe el `GITHUB_TOKEN` al mínimo necesario y revisa políticas de Actions. El workflow ya declara `contents: read`.
6. En `Settings → Security` activa **Dependabot alerts**, **Secret scanning** (según disponibilidad) y **Private vulnerability reporting** para recibir reportes privados.
7. Considera **Code scanning con CodeQL (default setup)** en repositorios donde esté disponible. No se ha incluido un workflow de CodeQL para no suponer que todas las cuentas tienen acceso ni duplicar el análisis predeterminado.
8. Verifica `Insights → Community Standards`: README, LICENSE, CONTRIBUTING, CODE_OF_CONDUCT, SECURITY e ISSUE_TEMPLATE deberían estar presentes. Los porcentajes y secciones exactas dependen de GitHub.
9. En releases, publica compilaciones validadas manualmente y hashes; no almacenes `.exe` en el código Git.
10. Si utilizas etiquetas automáticas en formularios, crea `bug`, `enhancement`, `compatibility` y `dependencies` en la sección **Labels**. Si no existen, GitHub puede no aplicar las etiquetas como esperas.

Documentación oficial:

- https://docs.github.com/en/repositories/creating-and-managing-repositories/best-practices-for-repositories
- https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/about-community-profiles-for-public-repositories
- https://docs.github.com/en/actions/reference/security/secure-use
- https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/configure-version-updates
