variable "project_id" {
  description = "ID del proyecto de Google Cloud donde se desplegarán los servicios."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id debe ser un ID de proyecto de Google Cloud válido de entre 6 y 30 caracteres."
  }
}

variable "region" {
  description = "Región de Google Cloud para todos los servicios de Cloud Run."
  type        = string

  validation {
    condition     = can(regex("^[a-z]+-[a-z]+[0-9]+$", var.region))
    error_message = "region debe ser una región válida de Google Cloud, como us-central1."
  }
}

variable "statistics_image" {
  description = "Referencia inmutable de la imagen de contenedor de la API de Estadísticas, preferentemente fijada por resumen criptográfico."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9.-]+/.+[^/]$", var.statistics_image))
    error_message = "statistics_image debe ser una referencia completa de imagen de contenedor."
  }
}

variable "matrix_image" {
  description = "Referencia inmutable de la imagen de contenedor de la API de Matrices, preferentemente fijada por resumen criptográfico."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9.-]+/.+[^/]$", var.matrix_image))
    error_message = "matrix_image debe ser una referencia completa de imagen de contenedor."
  }
}

variable "web_image" {
  description = "Referencia inmutable de la imagen de contenedor web, preferentemente fijada por resumen criptográfico."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9.-]+/.+[^/]$", var.web_image))
    error_message = "web_image debe ser una referencia completa de imagen de contenedor."
  }
}

variable "jwt_issuer" {
  description = "Declaración de emisor usada al emitir y validar los JWT de la aplicación."
  type        = string

  validation {
    condition     = trimspace(var.jwt_issuer) != ""
    error_message = "jwt_issuer no debe estar vacío."
  }
}

variable "jwt_audience" {
  description = "Declaración de audiencia usada al emitir y validar los JWT de la aplicación."
  type        = string

  validation {
    condition     = trimspace(var.jwt_audience) != ""
    error_message = "jwt_audience no debe estar vacío."
  }
}

variable "allowed_cors_origin" {
  description = "Origen explícito del navegador permitido por la API de Estadísticas; se rechazan comodines y rutas."
  type        = string

  validation {
    condition     = var.allowed_cors_origin != "*" && can(regex("^https?://[^/[:space:]]+$", var.allowed_cors_origin))
    error_message = "allowed_cors_origin debe ser un único origen HTTP(S) explícito, sin ruta ni barra final."
  }
}

variable "rsa_private_key_secret_id" {
  description = "ID del secreto existente en Secret Manager que contiene la clave privada RSA en formato PEM."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_-]{1,255}$", var.rsa_private_key_secret_id))
    error_message = "rsa_private_key_secret_id debe ser el ID de un secreto de Secret Manager, no el valor de un secreto."
  }
}

variable "rsa_public_key_secret_id" {
  description = "ID del secreto existente en Secret Manager que contiene la clave pública RSA en formato PEM."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_-]{1,255}$", var.rsa_public_key_secret_id))
    error_message = "rsa_public_key_secret_id debe ser el ID de un secreto de Secret Manager, no el valor de un secreto."
  }
}

variable "demo_username_secret_id" {
  description = "ID del secreto existente en Secret Manager que contiene el nombre de usuario de demostración."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_-]{1,255}$", var.demo_username_secret_id))
    error_message = "demo_username_secret_id debe ser el ID de un secreto de Secret Manager, no el valor de un secreto."
  }
}

variable "demo_password_secret_id" {
  description = "ID del secreto existente en Secret Manager que contiene la contraseña de demostración."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_-]{1,255}$", var.demo_password_secret_id))
    error_message = "demo_password_secret_id debe ser el ID de un secreto de Secret Manager, no el valor de un secreto."
  }
}
