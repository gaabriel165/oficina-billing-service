variable "region" {
  description = "AWS region to provision into"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Name of the project shared by every repository"
  type        = string
  default     = "oficina-api"
}

variable "service_name" {
  description = "Name prefix applied to the resources of this microservice"
  type        = string
  default     = "oficina-billing-service"
}

variable "state_bucket" {
  description = "S3 bucket holding the Terraform state of the other repositories"
  type        = string
  default     = "oficina-api-terraform-state-728750563430"
}

variable "kubernetes_namespace" {
  description = "Namespace of the workload, trusted by the pod role"
  type        = string
  default     = "oficina-billing"
}

variable "commands_queue_name" {
  description = "Queue with the saga commands consumed by this microservice"
  type        = string
  default     = "oficina-billing-commands"
}

variable "saga_replies_queue_name" {
  description = "Queue owned by the service order microservice that receives the saga replies"
  type        = string
  default     = "oficina-saga-replies"
}

variable "max_receive_count" {
  description = "Deliveries of a command before it is moved to the dead-letter queue"
  type        = number
  default     = 5
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.micro"
}

variable "db_engine_version" {
  description = "PostgreSQL engine version for RDS"
  type        = string
  default     = "16.15"
}

variable "db_name" {
  description = "Initial database name"
  type        = string
  default     = "oficina_billing"
}

variable "db_username" {
  description = "Master username for RDS"
  type        = string
  default     = "postgres"
}

variable "db_allocated_storage" {
  description = "Storage size in GiB"
  type        = number
  default     = 20
}

variable "db_backup_retention_days" {
  description = "Automated backup retention in days"
  type        = number
  default     = 1
}

variable "alert_email" {
  description = "E-mail subscribed to the alerts topic. Leave empty to create the topic without subscription"
  type        = string
  default     = ""
}
