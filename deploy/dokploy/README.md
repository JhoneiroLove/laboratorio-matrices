# Despliegue con Dokploy

Esta opción despliega la plataforma completa como una aplicación Docker Compose administrada por Dokploy. Dokploy ya incluye Traefik, dominios y certificados TLS, por lo que no se agrega Caddy ni se publican puertos del servidor manualmente.

## Topología

```text
Internet
   |
Traefik de Dokploy (HTTPS)
   |
web:8080 (Vue + Nginx)
   |
matrix-api-internal:8081 (API de Matrices)
   |
statistics-api-internal:3000 (API de Estadísticas)
```

Sólo `web` se conecta a la red externa `dokploy-network`. Las dos API se comunican por la red privada `private` y no deben tener dominios ni puertos públicos. Los alias explícitos `matrix-api-internal` y `statistics-api-internal` mantienen estable la resolución DNS interna y evitan colisiones con otros proyectos conectados a la red compartida de Dokploy.

## Requisitos previos

- Dokploy funcionando en la VPS.
- Puertos `80` y `443` disponibles para Traefik.
- Un dominio o subdominio con registro `A` apuntando a la IP pública de la VPS.
- Repositorio accesible desde Dokploy mediante GitHub o Git.

## Crear la aplicación

1. En Dokploy, creá un proyecto y dentro de él una aplicación de tipo **Docker Compose**.
2. Elegí GitHub o Git como proveedor y seleccioná este repositorio y la rama de despliegue.
3. Indicá `compose.dokploy.yaml` como ruta del archivo Compose.
4. En **Environment**, cargá las variables indicadas abajo. Dokploy guarda ese contenido como `.env`; el Compose referencia explícitamente cada variable mediante `environment`.
5. Guardá la configuración y ejecutá el primer despliegue.

No agregues `container_name`: Dokploy necesita administrar los nombres y el ciclo de vida de los servicios.

## Variables de entorno

Usá `deploy/dokploy/.env.example` como referencia y configurá, como mínimo:

```dotenv
PUBLIC_ORIGIN=https://matrices.example.com
JWT_ISSUER=https://matrices.example.com
JWT_AUDIENCE=matrix-platform
DEMO_USERNAME=administrador
DEMO_PASSWORD='una-contrasena-larga-generada-aleatoriamente'
```

Reglas importantes:

- `PUBLIC_ORIGIN` debe coincidir exactamente con el origen público, sin barra final.
- `JWT_ISSUER` puede usar el mismo origen público.
- `DEMO_PASSWORD` no debe reutilizar credenciales personales ni los valores locales del README principal.
- Escribí `DEMO_PASSWORD` entre comillas simples en el editor Environment. Así, caracteres como `$` se conservan literalmente y Compose no intenta interpretarlos como otra variable.
- Las variables opcionales ya tienen valores conservadores en el Compose.

## Configurar dominio y TLS

Después del primer despliegue:

1. Abrí la sección **Domains** de la aplicación Compose.
2. Creá un dominio para el servicio `web`.
3. Seleccioná el puerto interno `8080`.
4. Configurá la ruta `/`.
5. Habilitá HTTPS y el certificado Let's Encrypt administrado por Dokploy.
6. Guardá el dominio y ejecutá **Deploy** nuevamente. Dokploy materializa la configuración del dominio durante el despliegue; sin este paso, Traefik todavía no tendrá el nuevo enrutador.

No crees dominios para `backend` ni `statistics-api`. Nginx reenvía `/api/*` y `/health/*` hacia `backend` dentro de Docker.

## Persistencia y copias de seguridad

El volumen Docker `jwt-keys` guarda el par RSA. El servicio `jwt-keygen` lo crea únicamente cuando no existe y termina con código `0`; ese estado detenido es esperado.

En **Backups → Volume Backups** de Dokploy, programá una copia del volumen cuyo nombre sigue el patrón:

```text
{nombre-de-la-aplicacion}_jwt-keys
```

Ese volumen contiene la clave privada JWT y su copia debe almacenarse cifrada. La aplicación no persiste matrices ni estadísticas, por lo que no existen otros datos de negocio que respaldar.

No renombres la aplicación Compose sin planificar la migración del volumen: cambiar el nombre puede crear otro volumen y, por lo tanto, otro par de claves. Si las claves cambian, todos los tokens existentes quedan invalidados.

### Restaurar las claves después de perder la VPS

1. Creá en Dokploy la aplicación Compose con exactamente el mismo nombre anterior, pero no la despliegues todavía.
2. Si ya se ejecutó un despliegue accidental, detené todos los servicios y eliminá el volumen nuevo `{nombre-de-la-aplicacion}_jwt-keys`; no puede restaurarse una copia sobre un volumen existente y en uso.
3. Restaurá el backup usando como destino el nombre físico exacto `{nombre-de-la-aplicacion}_jwt-keys`.
4. Confirmá que el volumen restaurado existe y recién entonces ejecutá **Deploy**.
5. Verificá `/health/ready` y un inicio de sesión. No elimines el volumen durante actualizaciones o rollback normales.

Si la aplicación arranca antes de restaurar, `jwt-keygen` generará otro par RSA. Eso no pierde matrices porque no se persisten, pero invalida todos los JWT emitidos con las claves anteriores.

## Verificación posterior

```bash
curl --fail https://matrices.example.com/health/live
curl --fail https://matrices.example.com/health/ready
```

Ambas rutas deben devolver:

```json
{"status":"ok"}
```

Luego iniciá sesión desde el navegador y procesá una matriz rectangular. También podés ejecutar el E2E desde una estación con Node.js:

```bash
E2E_BASE_URL=https://matrices.example.com \
E2E_SKIP_DIRECT_STATISTICS=true \
DEMO_USERNAME=administrador \
DEMO_PASSWORD='la-misma-contrasena-configurada-en-dokploy' \
node --test tests/e2e/process-flow.test.mjs
```

La API de Estadísticas no está expuesta en producción. `E2E_SKIP_DIRECT_STATISTICS=true` omite únicamente sus pruebas HTTP directas; el flujo público sigue comprobando que la API de Matrices la invoque correctamente. Las pruebas normales desde el navegador tampoco necesitan acceso directo a Estadísticas.

## Actualizaciones y rollback

- Configurá el despliegue automático de Dokploy si querés publicar cada cambio de la rama seleccionada.
- Para una entrega controlada, usá etiquetas Git o una rama de publicación y desplegá manualmente desde Dokploy.
- Antes de actualizar, comprobá que CI haya aprobado pruebas, contratos y E2E.
- Para volver atrás, seleccioná un commit o etiqueta anterior y ejecutá un nuevo despliegue. El volumen `jwt-keys` no debe eliminarse durante el rollback.
- Revisá en Dokploy el estado, consumo y logs de `web`, `backend` y `statistics-api`. `jwt-keygen` debe permanecer finalizado con código `0`.

## Operación segura

- Mantené Dokploy, Docker y el sistema operativo actualizados.
- Permití en el firewall únicamente SSH restringido, `80` y `443`; no publiques `3000`, `8080` ni `8081`.
- Protegé la cuenta de Dokploy con una contraseña robusta y segundo factor si está disponible.
- Rotá las credenciales de demostración desde **Environment** y redesplegá.
- Conservá al menos una copia cifrada y probada del volumen de claves.
