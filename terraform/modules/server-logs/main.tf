# CloudWatch Logs for the server container (#94). The Docker awslogs driver on
# the EC2 instance writes to this group (see maintenance/ec2/
# docker-compose.cloudwatch.yml). The group is created here, not by the
# driver, so retention is always set and the instance role can stay limited to
# logs:CreateLogStream / logs:PutLogEvents on this one group.
#
# The server logs structured JSON (slog), so error lines are matched with a
# JSON metric filter on the level field.

locals {
  log_group_name = var.log_group_name != "" ? var.log_group_name : "/e2eetext/${var.environment}/server"
}

resource "aws_cloudwatch_log_group" "server" {
  name              = local.log_group_name
  retention_in_days = var.retention_in_days
  kms_key_id        = var.kms_key_id != "" ? var.kms_key_id : null

  tags = var.tags
}

# One metric per environment (dev and prod share the namespace). No dimensions:
# CloudWatch does not allow a default value together with dimensions, and the
# default of 0 keeps the metric continuous so the alarm can recover to OK.
locals {
  error_metric_name = "ServerErrorLogs-${var.environment}"
}

resource "aws_cloudwatch_log_metric_filter" "errors" {
  count = var.alarm_enabled ? 1 : 0

  name           = "${var.name}-server-errors"
  log_group_name = aws_cloudwatch_log_group.server.name
  pattern        = "{ $.level = \"ERROR\" }"

  metric_transformation {
    name          = local.error_metric_name
    namespace     = var.metrics_namespace
    value         = "1"
    default_value = "0"
  }
}

resource "aws_sns_topic" "alarm" {
  count = var.alarm_enabled ? 1 : 0

  name = "${var.name}-server-errors"
  tags = var.tags
}

resource "aws_sns_topic_subscription" "email" {
  count = var.alarm_enabled && var.alarm_email != "" ? 1 : 0

  topic_arn = aws_sns_topic.alarm[0].arn
  protocol  = "email"
  endpoint  = var.alarm_email
}

resource "aws_cloudwatch_metric_alarm" "errors" {
  count = var.alarm_enabled ? 1 : 0

  alarm_name          = "${var.name}-server-errors"
  alarm_description   = "Server logged at least ${var.alarm_threshold} error-level line(s) in ${var.alarm_period_seconds} seconds (log group ${local.log_group_name})."
  namespace           = var.metrics_namespace
  metric_name         = local.error_metric_name
  statistic           = "Sum"
  period              = var.alarm_period_seconds
  evaluation_periods  = 1
  threshold           = var.alarm_threshold
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.alarm[0].arn]
  ok_actions    = [aws_sns_topic.alarm[0].arn]

  tags = var.tags

  depends_on = [aws_cloudwatch_log_metric_filter.errors]
}
