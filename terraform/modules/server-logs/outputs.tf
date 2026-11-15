output "log_group_name" {
  description = "Name of the server log group."
  value       = aws_cloudwatch_log_group.server.name
}

output "log_group_arn" {
  description = "ARN of the server log group."
  value       = aws_cloudwatch_log_group.server.arn
}

output "alarm_topic_arn" {
  description = "SNS topic the error alarm publishes to, or null when the alarm is disabled."
  value       = var.alarm_enabled ? aws_sns_topic.alarm[0].arn : null
}
