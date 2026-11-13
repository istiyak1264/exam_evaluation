variable "project" {
  description = "Short name used to prefix every resource."
  type        = string
  default     = "exam-evaluation"
}

variable "region" {
  type    = string
  default = "eu-west-1"
}

variable "admin_email" {
  description = "Login email for the bootstrap administrator (required; not stored in code)."
  type        = string
}

variable "admin_name" {
  description = "Display name for the bootstrap administrator."
  type        = string
  default     = "Administrator"
}

variable "vpc_cidr" {
  type    = string
  default = "10.42.0.0/16"
}

variable "azs" {
  description = "Two AZs are required for the RDS subnet group."
  type        = list(string)
  default     = ["eu-west-1a", "eu-west-1b"]
}

variable "web_desired_count" {
  type    = number
  default = 2
}

variable "backend_desired_count" {
  type    = number
  default = 2
}

variable "backend_cpu" {
  type    = number
  default = 512
}

variable "backend_memory" {
  type    = number
  default = 1024
}

variable "db_instance_class" {
  type    = string
  default = "db.t4g.small"
}

variable "db_name" {
  type    = string
  default = "exam_eval"
}

variable "db_username" {
  type    = string
  default = "exam"
}

variable "db_multi_az" {
  description = "Enable for production. Costs roughly double."
  type        = bool
  default     = false
}

variable "db_deletion_protection" {
  type    = bool
  default = true
}

variable "admin_cidr" {
  description = "Optional CIDR allowed to reach the ALB (empty = whole internet)."
  type        = string
  default     = ""
}

variable "acm_certificate_arn" {
  description = "Certificate for HTTPS on the ALB. Empty keeps the listener HTTP-only."
  type        = string
  default     = ""
}

variable "image_tag" {
  description = "Image tag deployed from ECR (set by CI to the commit SHA)."
  type        = string
  default     = "latest"
}

variable "log_retention_days" {
  type    = number
  default = 90
}

variable "tags" {
  type = map(string)
  default = {
    Project   = "exam-evaluation"
    Owner     = "examinations-office"
    ManagedBy = "terraform"
  }
}
