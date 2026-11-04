output "alb_dns_name" {
  description = "Point DNS at this (CNAME) once the certificate is issued."
  value       = aws_lb.main.dns_name
}

output "ecr_backend" {
  value = aws_ecr_repository.backend.repository_url
}

output "ecr_web" {
  value = aws_ecr_repository.web.repository_url
}

output "db_endpoint" {
  value     = aws_db_instance.main.address
  sensitive = false
}

output "scripts_bucket" {
  value = aws_s3_bucket.scripts.id
}

output "ecs_cluster" {
  value = aws_ecs_cluster.main.name
}

output "app_secret_arn" {
  value = aws_secretsmanager_secret.app.arn
}

output "kms_key_arn" {
  value = aws_kms_key.scripts.arn
}
