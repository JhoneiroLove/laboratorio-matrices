# Estrategia de pruebas

## Pruebas de contrato

- Validar ambos documentos OpenAPI 3.1 y resolver todas las referencias externas de JSON Schema.
- Ejercitar cada código de estado y verificar los tipos de contenido y campos obligatorios de RFC 9457.
- Verificar que los puntos de conexión de salud no requieran un token de portador y que todos los demás puntos de conexión ajenos a la autenticación sí lo requieran.
- Ejecutar pruebas de proveedor y consumidor para la invocación síncrona de la API de Matrices a la API de Estadísticas.

## Pruebas numéricas

- Usar matrices cuadradas, altas, anchas, de un solo elemento, nulas, negativas y fraccionarias.
- Para la rotación, comprobar las dimensiones y las posiciones exactas de los elementos.
- Para QR, comprobar las dimensiones, `Q^T Q ~= I`, `Q R ~= A` y que `R` sea trapezoidal superior dentro del épsilon configurado (cuyo valor predeterminado actual es `1e-10`); no comparar matrices de punto flotante mediante igualdad exacta.
- Para `/process`, comprobar que tanto la rotación como QR usen de forma independiente la entrada original y que la API de Estadísticas reciba exactamente `rotated`, `Q` y `R` con las matrices de salida correspondientes.
- Incluir matrices casi diagonales y no cuadradas en torno al límite del épsilon.
- Verificar los conteos y promedios de los resúmenes en matrices con distintas dimensiones.

## Pruebas de resiliencia

- Cubrir matrices malformadas, vacías y con filas de distinta longitud; nombres de matriz duplicados; tokens no válidos; tiempos de espera agotados en el servicio ascendente; detalles del problema del servicio ascendente; y respuestas exitosas malformadas del servicio ascendente.
- Probar los límites configurados en los valores activos, por debajo y por encima de ellos, sin tratar los valores predeterminados actuales como constantes inmutables del contrato.
- Verificar la propagación del ID de solicitud entre ambos servicios y en las respuestas de problema.

El servicio en Go usa pruebas basadas en tablas de la biblioteca estándar. La API de Estadísticas y Vue usan Vitest, con Supertest para la integración HTTP. `tests/e2e/process-flow.test.mjs` ejercita la autenticación y el flujo completo de Vue como puerta de enlace hacia Go y Node.js sobre Docker Compose. `.github/workflows/ci.yml` ejecuta el análisis de contratos, las pruebas unitarias y de integración, las comprobaciones estáticas y el flujo de aceptación de la pila completa.
