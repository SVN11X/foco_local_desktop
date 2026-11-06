# Estrategia de pruebas

## Automatizadas (CI)

En Ubuntu y Windows:

```powershell
go test -count=1 ./...
go vet ./...
go mod verify
```

Además, el workflow valida `gofmt` y compila `FocoLocal.exe` para Windows x64. Estas pruebas **no demuestran** compatibilidad de la interfaz con navegadores reales.

## Validación manual obligatoria antes de publicar una release estable

| Escenario | Resultado que debe comprobarse |
| --- | --- |
| Chrome en Windows 10/11 | Reconoce dominio y contabiliza tiempo visible |
| Edge en Windows 10/11 | Comparte cuota con Chrome |
| Firefox | Registra si UI Automation expone dirección; si no, informar limitación |
| Brave / Opera / Vivaldi | Validar cada versión sin asumir compatibilidad |
| Ventana minimizada / navegador en segundo plano | No contabiliza como uso activo |
| Cambio de dominio / pestaña | No arrastra segundos a un dominio distinto |
| Reinicio / suspensión del equipo | Conserva datos y no contabiliza suspensión |
| Playwright con perfil propio | No cerrar ni superponer el navegador automatizado |
| Selenium, Python, Node.js, VS Code, Git | No cortar conexiones ni cerrar procesos |
| Navegador controlado desde una ventana existente | Documentar riesgo de cubierta visual |
| Alta DPI, escala 125–200 %, dos monitores | La cubierta no invade otras ventanas |
| Navegación de teclado / lector de pantalla | Registrar barreras y correcciones |
| Sin permisos de administrador | Funcionamiento normal para usuario estándar |

Anota **probado/no probado/falla** por cada navegador, build y versión de Windows. Nunca marques soporte como universal basándote solo en la compilación.

## Privacidad del informe

Utiliza dominios de prueba públicos (`example.com`) y no pegues `%APPDATA%` ni capturas con historial personal. No adjuntes tokens, contraseñas, cookies o datos de navegación privada.

## Prueba manual: monitor flotante sin robar foco (v0.2.1)

Configura un límite de dos minutos para YouTube, pulsa **Probar con monitor flotante** y deja Edge o Chrome como ventana activa. Verifica que el monitor muestre el dominio y los segundos acumulados y que aparezca la cubierta al agotarse el tiempo. Si el monitor indica **No se reconoce la barra de direcciones (UIA)**, captura el mensaje y el nombre del navegador. Repite con otra ventana para confirmar que no se contabiliza actividad en segundo plano ni se intervienen los procesos automatizados. Consulta [GUIA_PRUEBA_WINDOWS.md](GUIA_PRUEBA_WINDOWS.md).
