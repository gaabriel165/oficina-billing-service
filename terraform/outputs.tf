output "commands_queue_url" {
  description = "URL of the queue with the saga commands"
  value       = aws_sqs_queue.commands.id
}

output "commands_dlq_name" {
  description = "Dead-letter queue of the saga commands, watched by the alerts"
  value       = aws_sqs_queue.commands_dlq.name
}

output "pod_role_arn" {
  description = "Role annotated on the Kubernetes service account"
  value       = aws_iam_role.pod.arn
}

output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint (host)"
  value       = aws_db_instance.this.address
}

output "database_url_parameter_name" {
  description = "SSM parameter holding the full connection string"
  value       = aws_ssm_parameter.database_url.name
}
