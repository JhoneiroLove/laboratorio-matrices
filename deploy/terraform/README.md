# Despliegue en Google Cloud Run

Esta raíz de Terraform despliega las imágenes preconstruidas de la API de Estadísticas, la API de Matrices y el servicio web en Google Cloud Run v2. No crea bases de datos ni recursos de Kafka o Redis. Terraform administra la configuración de los servicios y el acceso a secretos, pero nunca lee ni almacena sus cargas.

## Requisitos previos

- Terraform `>= 1.7` y Google Cloud CLI autenticado con Application Default Credentials.
- Un proyecto de Google Cloud donde la identidad principal del despliegue pueda habilitar API, crear cuentas de servicio y servicios de Cloud Run, y administrar IAM en los secretos referenciados.
- Cuatro secretos existentes en Secret Manager con versiones habilitadas: una clave privada RSA, su clave pública RSA, un nombre de usuario de demostración y una contraseña de demostración.
- Tres imágenes de contenedor preconstruidas accesibles para el agente de servicio de Cloud Run. Se recomiendan referencias de Artifact Registry fijadas por resumen criptográfico.
- Un almacenamiento remoto del estado de Terraform configurado por el operador para uso compartido o en producción. Esta raíz no prescribe deliberadamente un mecanismo de almacenamiento.

## Publicar imágenes preconstruidas

Creá una vez un repositorio Docker de Artifact Registry si todavía no existe:

```sh
gcloud artifacts repositories create matrix --repository-format=docker --location=us-central1 --project=YOUR_PROJECT_ID
gcloud auth configure-docker us-central1-docker.pkg.dev
```

Etiquetá y publicá las imágenes locales ya construidas sin volver a construirlas:

```sh
docker tag statistics:release us-central1-docker.pkg.dev/YOUR_PROJECT_ID/matrix/statistics:release
docker tag matrix:release us-central1-docker.pkg.dev/YOUR_PROJECT_ID/matrix/matrix:release
docker tag web:release us-central1-docker.pkg.dev/YOUR_PROJECT_ID/matrix/web:release
docker push us-central1-docker.pkg.dev/YOUR_PROJECT_ID/matrix/statistics:release
docker push us-central1-docker.pkg.dev/YOUR_PROJECT_ID/matrix/matrix:release
docker push us-central1-docker.pkg.dev/YOUR_PROJECT_ID/matrix/web:release
```

Obtené cada resumen criptográfico publicado con `gcloud artifacts docker images describe IMAGE:TAG --format='value(image_summary.digest)'` y luego usá `IMAGE@sha256:...` en `terraform.tfvars`.

## Crear versiones de secretos

Creá una vez los contenedores de secretos si es necesario:

```sh
gcloud secrets create matrix-jwt-private-key --replication-policy=automatic --project=YOUR_PROJECT_ID
gcloud secrets create matrix-jwt-public-key --replication-policy=automatic --project=YOUR_PROJECT_ID
gcloud secrets create matrix-demo-username --replication-policy=automatic --project=YOUR_PROJECT_ID
gcloud secrets create matrix-demo-password --replication-policy=automatic --project=YOUR_PROJECT_ID
```

Agregá las cargas desde archivos locales o la entrada estándar. No incluyas cargas en `.tfvars` ni en recursos de Terraform:

```sh
gcloud secrets versions add matrix-jwt-private-key --data-file=jwt-private.pem --project=YOUR_PROJECT_ID
gcloud secrets versions add matrix-jwt-public-key --data-file=jwt-public.pem --project=YOUR_PROJECT_ID
printf '%s' "$DEMO_USERNAME" | gcloud secrets versions add matrix-demo-username --data-file=- --project=YOUR_PROJECT_ID
printf '%s' "$DEMO_PASSWORD" | gcloud secrets versions add matrix-demo-password --data-file=- --project=YOUR_PROJECT_ID
```

Terraform otorga a la identidad de la API de Estadísticas acceso únicamente a la clave pública. La identidad de la API de Matrices puede acceder a ambas claves y a las dos credenciales de demostración. La identidad del servicio web no recibe roles del proyecto. Las referencias a secretos usan la versión habilitada `latest`; desplegá una nueva revisión de Cloud Run después de rotar un secreto para que todas las instancias usen de forma coherente la nueva carga.

## Desplegar

```sh
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform fmt -check
terraform validate
terraform plan -out=deployment.tfplan
terraform apply deployment.tfplan
```

Establecé `allowed_cors_origin` con el origen web definitivo. Debe ser un origen explícito sin ruta ni barra final. Terraform inyecta la URL de la API de Estadísticas en la API de Matrices como `ANALYTICS_BASE_URL` y la URL de la API de Matrices en el servicio web como `MATRIX_API_URL`. Esta última no tiene una barra final de forma intencional para que la plantilla de Nginx de la imagen preserve `/api/...` al actuar como servidor intermediario.

## Exposición y refuerzo

Los tres servicios permiten la invocación de Cloud Run sin autenticación para el desafío técnico o la demostración. La validación de JWT en la aplicación protege los puntos de conexión de negocio, pero Cloud Run sigue recibiendo solicitudes sin autenticar y CORS no constituye un límite de autorización. Los puntos de conexión de salud permanecen públicos de forma intencional.

Para una iteración de refuerzo en producción, eliminá la invocación pública de la API de Estadísticas, otorgá su permiso `roles/run.invoker` únicamente a la identidad de ejecución de la API de Matrices y agregá tokens de identidad de servicio firmados por Google a las solicitudes de la API de Matrices a la API de Estadísticas. La aplicación actual reenvía el JWT del usuario y todavía no emite ese token de identidad de servicio, por lo que hacer que la API de Estadísticas sea privada antes de ese cambio interrumpiría la ruta de llamadas.
