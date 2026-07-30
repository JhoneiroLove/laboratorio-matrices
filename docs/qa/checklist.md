# Lista de verificación de QA

## Contrato

- [x] Los documentos OpenAPI 3.1 superan la validación con las referencias externas resueltas.
- [x] Los ejemplos y las cargas de la implementación coinciden con los esquemas JSON compartidos.
- [x] El comportamiento de los puntos de conexión protegidos y públicos coincide con las declaraciones de seguridad.
- [x] Todas las respuestas de error usan `application/problem+json` y los campos de RFC 9457.

## Comportamiento

- [x] La rotación en sentido horario es correcta para matrices cuadradas y rectangulares.
- [x] La QR reducida de Householder satisface las invariantes dimensionales y basadas en el épsilon.
- [x] La API de Estadísticas devuelve el mínimo, máximo, suma, promedio y cantidad de elementos a nivel global y por matriz.
- [x] `diagonal` por matriz y el agregado `anyDiagonal` cumplen las reglas de matrices cuadradas y del épsilon.
- [x] `/process` propaga un ID de solicitud y gestiona los tiempos de espera agotados y errores de la API de Estadísticas.

## Requisitos no funcionales

- [x] Los límites configurables activos de matrices y cargas están documentados, son positivos y se aplican sin presentar los valores predeterminados como garantías inmutables de la API.
- [x] Los registros omiten credenciales, tokens de portador y cuerpos completos de matrices.
- [ ] El comportamiento de disponibilidad y preparación se verifica durante errores de dependencias y el apagado.
- [x] `/process` obtiene la rotación y la QR reducida de forma independiente a partir de la entrada original, y envía exactamente `rotated`, `Q` y `R` a la API de Estadísticas.
