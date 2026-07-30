# Generación reproducible del PDF

El repositorio almacena las fuentes Markdown, no los PDF generados. Una versión puede crear un PDF en `docs/generated/`, un directorio ignorado por Git.

## Cadena de herramientas con versiones fijadas

- Pandoc `3.6.4`
- Tectonic `0.15.0`

El repositorio también puede usar la imagen de contenedor `pandoc/latex:3.6.4`, cuya versión está fijada, cuando el equipo anfitrión no dispone de Pandoc y Tectonic. El resumen criptográfico verificado de la imagen es `sha256:505c16cb042a5cb2e7825da6ea7368278e8eb8fc3aff06062e80f790ca00dec8`.

Instalá exactamente esas versiones desde artefactos de publicación confiables y registrá sus sumas de comprobación en el trabajo de publicación. Ejecutá el comando desde la raíz del repositorio con una configuración regional, zona horaria y marca de tiempo de origen fijas:

```sh
mkdir -p docs/generated
export LANG=C.UTF-8
export TZ=UTC
export SOURCE_DATE_EPOCH=1785369600
pandoc \
  README.md \
  docs/architecture/overview.md \
  docs/adr/0001-qr-and-rotation-order.md \
  docs/adr/0002-synchronous-http-and-no-database.md \
  docs/security/baseline.md \
  docs/testing/strategy.md \
  docs/deployment/guide.md \
  deploy/dokploy/README.md \
  docs/qa/checklist.md \
  docs/qa/ejecucion-local-2026-07-30.md \
  --from=gfm \
  --standalone \
  --pdf-engine=tectonic \
  --metadata title="Plataforma de procesamiento de matrices" \
  --metadata date="2026-07-30" \
  --output docs/generated/matrix-platform.pdf
```

El orden de entrada es intencional y debe permanecer explícito. La integración continua debería publicar el PDF y su suma de comprobación SHA-256 como artefactos de la versión, en lugar de versionar el archivo binario. Un trabajo de publicación completamente hermético debería fijar por resumen criptográfico la imagen del sistema operativo y los artefactos de las herramientas; esa infraestructura queda deliberadamente fuera de esta base dedicada a la documentación.

## Generación mediante un contenedor

Desde la raíz del repositorio, el siguiente comando usa las mismas fuentes ordenadas sin instalar dependencias en el equipo anfitrión:

```sh
docker run --rm \
  --volume "$PWD:/data" \
  --workdir /data \
  pandoc/latex:3.6.4 \
  README.md \
  docs/architecture/overview.md \
  docs/adr/0001-qr-and-rotation-order.md \
  docs/adr/0002-synchronous-http-and-no-database.md \
  docs/security/baseline.md \
  docs/testing/strategy.md \
  docs/deployment/guide.md \
  deploy/dokploy/README.md \
  docs/qa/checklist.md \
  docs/qa/ejecucion-local-2026-07-30.md \
  --from=gfm \
  --standalone \
  --pdf-engine=xelatex \
  --metadata title="Plataforma de procesamiento de matrices" \
  --metadata date="2026-07-30" \
  --output docs/generated/matrix-platform.pdf
```

Quienes usen PowerShell pueden reemplazar `"$PWD:/data"` por `"${PWD}:/data"`. Verificá el artefacto con `sha256sum` o `Get-FileHash` antes de publicarlo.
