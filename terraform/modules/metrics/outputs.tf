output "function_name" {
  description = "Name of the metrics Lambda."
  value       = aws_lambda_function.this.function_name
}

output "function_arn" {
  description = "ARN of the metrics Lambda."
  value       = aws_lambda_function.this.arn
}

output "namespace" {
  description = "CloudWatch namespace the metrics are published to."
  value       = var.namespace
}

output "security_group_id" {
  description = "Security group ID of the Lambda."
  value       = aws_security_group.lambda.id
}
