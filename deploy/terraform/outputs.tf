output "statistics_url" {
  description = "URL pública de la API de Estadísticas en Cloud Run."
  value       = google_cloud_run_v2_service.statistics.uri
}

output "matrix_url" {
  description = "URL pública de la API de Matrices en Cloud Run."
  value       = google_cloud_run_v2_service.matrix.uri
}

output "web_url" {
  description = "URL pública del servicio web de Cloud Run."
  value       = google_cloud_run_v2_service.web.uri
}

output "runtime_service_accounts" {
  description = "Identidades de ejecución dedicadas asignadas a las revisiones de Cloud Run."
  value = {
    statistics = google_service_account.statistics.email
    matrix     = google_service_account.matrix.email
    web        = google_service_account.web.email
  }
}
