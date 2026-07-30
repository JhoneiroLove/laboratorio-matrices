locals {
  secret_ids = {
    private_key = var.rsa_private_key_secret_id
    public_key  = var.rsa_public_key_secret_id
    username    = var.demo_username_secret_id
    password    = var.demo_password_secret_id
  }
}

resource "google_project_service" "required" {
  for_each = toset([
    "artifactregistry.googleapis.com",
    "iam.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
  ])

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

resource "google_service_account" "statistics" {
  project      = var.project_id
  account_id   = "statistics-cloud-run"
  display_name = "Entorno de ejecución de la API de Estadísticas en Cloud Run"

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_service_account" "matrix" {
  project      = var.project_id
  account_id   = "matrix-cloud-run"
  display_name = "Entorno de ejecución de la API de Matrices en Cloud Run"

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

resource "google_service_account" "web" {
  project      = var.project_id
  account_id   = "web-cloud-run"
  display_name = "Entorno de ejecución web en Cloud Run"

  depends_on = [google_project_service.required["iam.googleapis.com"]]
}

data "google_secret_manager_secret" "existing" {
  for_each = local.secret_ids

  project   = var.project_id
  secret_id = each.value

  depends_on = [google_project_service.required["secretmanager.googleapis.com"]]
}

resource "google_secret_manager_secret_iam_member" "statistics_public_key" {
  project   = var.project_id
  secret_id = data.google_secret_manager_secret.existing["public_key"].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.statistics.email}"
}

resource "google_secret_manager_secret_iam_member" "matrix_secrets" {
  for_each = data.google_secret_manager_secret.existing

  project   = var.project_id
  secret_id = each.value.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.matrix.email}"
}

resource "google_cloud_run_v2_service" "statistics" {
  project             = var.project_id
  name                = "statistics-api"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    service_account                  = google_service_account.statistics.email
    timeout                          = "30s"
    max_instance_request_concurrency = 80

    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }

    volumes {
      name = "jwt-public-key"

      secret {
        secret       = data.google_secret_manager_secret.existing["public_key"].secret_id
        default_mode = 292

        items {
          version = "latest"
          path    = "key.pem"
        }
      }
    }

    containers {
      name  = "statistics"
      image = var.statistics_image

      ports {
        name           = "http1"
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
        cpu_idle          = true
        startup_cpu_boost = true
      }

      env {
        name  = "PORT"
        value = "8080"
      }

      env {
        name  = "JWT_PUBLIC_KEY_PATH"
        value = "/var/run/secrets/jwt-public/key.pem"
      }

      env {
        name  = "JWT_ISSUER"
        value = var.jwt_issuer
      }

      env {
        name  = "JWT_AUDIENCE"
        value = var.jwt_audience
      }

      env {
        name  = "CORS_ORIGINS"
        value = var.allowed_cors_origin
      }

      volume_mounts {
        name       = "jwt-public-key"
        mount_path = "/var/run/secrets/jwt-public"
      }

      startup_probe {
        initial_delay_seconds = 0
        timeout_seconds       = 2
        period_seconds        = 3
        failure_threshold     = 10

        http_get {
          path = "/health/ready"
          port = 8080
        }
      }

      liveness_probe {
        timeout_seconds   = 2
        period_seconds    = 10
        failure_threshold = 3

        http_get {
          path = "/health/live"
          port = 8080
        }
      }
    }
  }

  depends_on = [
    google_project_service.required["run.googleapis.com"],
    google_secret_manager_secret_iam_member.statistics_public_key,
  ]
}

resource "google_cloud_run_v2_service" "matrix" {
  project             = var.project_id
  name                = "matrix-api"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    service_account                  = google_service_account.matrix.email
    timeout                          = "60s"
    max_instance_request_concurrency = 40

    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }

    volumes {
      name = "jwt-private-key"

      secret {
        secret       = data.google_secret_manager_secret.existing["private_key"].secret_id
        default_mode = 292

        items {
          version = "latest"
          path    = "key.pem"
        }
      }
    }

    volumes {
      name = "jwt-public-key"

      secret {
        secret       = data.google_secret_manager_secret.existing["public_key"].secret_id
        default_mode = 292

        items {
          version = "latest"
          path    = "key.pem"
        }
      }
    }

    containers {
      name  = "matrix"
      image = var.matrix_image

      ports {
        name           = "http1"
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
        cpu_idle          = true
        startup_cpu_boost = true
      }

      env {
        name  = "ANALYTICS_BASE_URL"
        value = google_cloud_run_v2_service.statistics.uri
      }

      env {
        name  = "JWT_PRIVATE_KEY_PATH"
        value = "/var/run/secrets/jwt-private/key.pem"
      }

      env {
        name  = "JWT_PUBLIC_KEY_PATH"
        value = "/var/run/secrets/jwt-public/key.pem"
      }

      env {
        name  = "JWT_ISSUER"
        value = var.jwt_issuer
      }

      env {
        name  = "JWT_AUDIENCE"
        value = var.jwt_audience
      }

      env {
        name = "DEMO_USERNAME"

        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.existing["username"].secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "DEMO_PASSWORD"

        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.existing["password"].secret_id
            version = "latest"
          }
        }
      }

      volume_mounts {
        name       = "jwt-private-key"
        mount_path = "/var/run/secrets/jwt-private"
      }

      volume_mounts {
        name       = "jwt-public-key"
        mount_path = "/var/run/secrets/jwt-public"
      }

      startup_probe {
        initial_delay_seconds = 0
        timeout_seconds       = 2
        period_seconds        = 3
        failure_threshold     = 10

        http_get {
          path = "/health/ready"
          port = 8080
        }
      }

      liveness_probe {
        timeout_seconds   = 2
        period_seconds    = 10
        failure_threshold = 3

        http_get {
          path = "/health/live"
          port = 8080
        }
      }
    }
  }

  depends_on = [
    google_project_service.required["run.googleapis.com"],
    google_secret_manager_secret_iam_member.matrix_secrets,
  ]
}

resource "google_cloud_run_v2_service" "web" {
  project             = var.project_id
  name                = "matrix-web"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    service_account                  = google_service_account.web.email
    timeout                          = "30s"
    max_instance_request_concurrency = 80

    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }

    containers {
      name  = "web"
      image = var.web_image

      ports {
        name           = "http1"
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "256Mi"
        }
        cpu_idle          = true
        startup_cpu_boost = true
      }

      env {
        name  = "MATRIX_API_URL"
        value = google_cloud_run_v2_service.matrix.uri
      }

      startup_probe {
        initial_delay_seconds = 0
        timeout_seconds       = 2
        period_seconds        = 3
        failure_threshold     = 10

        http_get {
          path = "/"
          port = 8080
        }
      }

      liveness_probe {
        timeout_seconds   = 2
        period_seconds    = 10
        failure_threshold = 3

        http_get {
          path = "/"
          port = 8080
        }
      }
    }
  }

  depends_on = [google_project_service.required["run.googleapis.com"]]
}

resource "google_cloud_run_v2_service_iam_member" "public" {
  for_each = {
    statistics = google_cloud_run_v2_service.statistics.name
    matrix     = google_cloud_run_v2_service.matrix.name
    web        = google_cloud_run_v2_service.web.name
  }

  project  = var.project_id
  location = var.region
  name     = each.value
  role     = "roles/run.invoker"
  member   = "allUsers"
}
