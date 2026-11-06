# Guía de experiencia de usuario — FocoLocal v0.3.0

El hub utiliza una jerarquía clara, colores de alto contraste y componentes Win32 que no exigen instalación adicional. Las vistas reducen la carga cognitiva: **Resumen**, **Explorar sitios**, **Crear regla**, **Preferencias**.

- La selección de un sitio en la biblioteca **solo prepara un formulario**, nunca activa el bloqueo. Se guardan los cambios exclusivamente al pulsar **Guardar regla**.
- Los sitios están agrupados por categoría y llevan un identificador visual con colores/abreviaturas locales (no se descargan favicons ni contenido remoto).
- Las opciones temporales muestran la unidad correspondiente: minutos, horas, diario, semanal, mensual, todo el día, horario o ciclos. Los campos no aplicables aparecen deshabilitados.
- Los estados usan textos además de color (Activo, Agotado, Pausado), y el contador visual se actualiza cada segundo.
- Se mantienen teclado y orden natural de tabulación; los botones personalizados dibujan un borde de foco. Se utiliza la fuente Segoe UI disponible en Windows.
- En la segunda pantalla se ofrece un modo manual, nunca inferencia de mirada; la preferencia no se guarda entre reinicios y requiere ventana visible y actividad reciente.
- La pantalla de bloqueo es HTML local, accesible en tamaños variables, sin JavaScript, telemetría ni fuentes remotas. Ofrece actividades alternativas y explica el reinicio de la regla.
- En resoluciones reducidas se ajustan proporcionalmente controles y superficies; los textos más largos pueden necesitar validación adicional con escalado de Windows 125 %/150 %.

**Pendiente:** validar la interfaz directamente en Windows, lector de pantalla, alto contraste, accesibilidad de la lista de reglas, el icono del ejecutable y localización. Los marcadores gráficos de los sitios son iconos de identificación simples, no logotipos oficiales.
