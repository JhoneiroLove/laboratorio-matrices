# Línea base de seguridad

## Autenticación y autorización

- `/api/v1/auth/token` y las sondas de salud son públicos; las operaciones de matrices y estadísticas requieren un token de portador HTTP.
- Los tokens se firman y verifican únicamente con RS256. Ambas API validan el emisor, la audiencia, la expiración, el inicio de validez y el algoritmo permitido; solo la API de Matrices tiene acceso a la clave privada.
- El intercambio integrado de nombre de usuario y contraseña existe para el desafío técnico. Un sistema de producción debería delegar la autenticación en un proveedor OIDC y rotar las claves mediante el administrador de secretos de la plataforma.
- Actualmente, la API de Matrices reenvía el JWT de quien realiza la solicitud a la API de Estadísticas para que ambas API apliquen el requisito del desafío. Un despliegue privado reforzado de la API de Estadísticas también debería autenticar la identidad de carga de trabajo de la API de Matrices; consultá la guía de despliegue.
- Los errores de autenticación devuelven detalles genéricos conforme a RFC 9457 y un encabezado `WWW-Authenticate`, sin revelar la validez de las credenciales.
- La API de Matrices limita en memoria las solicitudes de tokens por dirección del cliente. Esto protege el despliegue del desafío, pero un despliegue de producción con escalado horizontal debería aplicar un límite compartido en la puerta de enlace o el proveedor de identidad.
- Ambas API aplican la misma tolerancia de reloj configurable, de 30 segundos de forma predeterminada, a las declaraciones temporales de los JWT.

## Entrada y transporte

- El tráfico de producción debe usar TLS, incluido el HTTP entre servicios.
- Docker Compose es una topología de desarrollo que solo usa HTTP y cuyos puertos publicados se vinculan a `127.0.0.1`; no debe exponerse como punto de conexión de producción.
- Las implementaciones deben aplicar límites positivos y configurables al cuerpo de las solicitudes, las dimensiones de las matrices y la cantidad de matrices y elementos para evitar el agotamiento de memoria y CPU. Los valores predeterminados actuales se documentan en el resumen de arquitectura; son opciones de despliegue, no garantías universales de la API.
- Rechazar matrices con filas de distinta longitud, filas vacías, números no finitos, campos inesperados y JSON malformado.
- Nunca registrar contraseñas, tokens de acceso ni cargas completas de matrices. Correlacionar los registros mediante los ID de solicitud.
- Vue conserva el token de acceso únicamente en memoria. Al actualizar o cerrar la página, el token se descarta.
- Fiber y Express establecen encabezados HTTP defensivos; el navegador accede a la API de Matrices a través de un servidor intermediario del mismo origen, por lo que no se necesita una política CORS permisiva.

## Operaciones

Las dependencias y las imágenes de contenedores tienen versiones fijadas y deberían analizarse en el flujo de publicación. Los contenedores de ejecución usan usuarios sin privilegios de `root`, y el entorno de ejecución de Go usa una imagen sin distribución (distroless). Los secretos deben permitir su rotación. Las respuestas de salud no exponen configuraciones ni credenciales de dependencias.
