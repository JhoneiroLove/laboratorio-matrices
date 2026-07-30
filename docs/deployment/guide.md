# Guía de despliegue

La aplicación admite dos destinos documentados:

- [`deploy/terraform`](../../deploy/terraform/README.md) despliega tres imágenes preconstruidas en Google Cloud Run v2.
- [`deploy/dokploy`](../../deploy/dokploy/README.md) despliega el Compose completo en una VPS administrada mediante Dokploy y Traefik.

Ninguna opción aprovisiona bases de datos ni infraestructura de Kafka o Redis.

## Topología de ejecución

- Desplegar el servicio web, la API de Matrices y la API de Estadísticas como cargas de trabajo independientes y sin estado, con cuentas de servicio de ejecución dedicadas.
- La configuración del desafío o demostración expone públicamente los tres servicios de Cloud Run. La validación de JWT en la aplicación protege los puntos de conexión de negocio; permitir la invocación pública de Cloud Run no constituye autenticación por sí mismo.
- Como refuerzo futuro para producción, hacer que la API de Estadísticas sea privada, permitir su invocación únicamente a la identidad de servicio de la API de Matrices y hacer que esta envíe un token de identidad de servicio firmado por Google, además de reenviar el JWT de la aplicación. La aplicación todavía no implementa ese token entre servicios.
- Configurar TLS, los tokens de portador, la URL base de la API de Estadísticas, los tiempos de espera de conexión y solicitud, y el apagado controlado mediante el entorno de ejecución.
- Configurar límites para el cuerpo de las solicitudes, las filas, las columnas y la cantidad de matrices y elementos según la CPU, la memoria y los objetivos de latencia disponibles. Los valores predeterminados actuales documentados son puntos de partida, no garantías estrictas de la plataforma.
- Escalar cada servicio de forma independiente. No depender del estado de sesión local.

## Sondas de salud

- `/health/live` informa si el proceso está activo y no debe depender de servicios descendentes.
- `/health/ready` informa si una instancia puede aceptar tráfico. La API de Matrices comprueba la API de Estadísticas con un tiempo de espera limitado; las plataformas deben usar una frecuencia moderada para no agravar una interrupción.
- Las sondas son públicas y no deben devolver información confidencial.

## Publicación

Validar los contratos, esquemas, pruebas, la política de dependencias y las comprobaciones de seguridad antes de publicar. Los despliegues progresivos deberían preservar la compatibilidad con el par consumidor/proveedor desplegado actualmente. Usar imágenes fijadas por resumen criptográfico y revisar `terraform plan` antes del despliegue; consultá el README de Terraform para conocer los procedimientos de publicación, creación de versiones de secretos y despliegue.

En Dokploy, sólo el servicio `web` debe recibir un dominio. Las API permanecen en una red privada de Compose y Nginx centraliza las rutas públicas `/api/*` y `/health/*`. El volumen de claves JWT debe incluirse en las copias de seguridad de volúmenes de Dokploy.
