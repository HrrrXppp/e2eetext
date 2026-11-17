variable "name" {
  description = "Name prefix (e.g. \"e2eetext-prod\"). Used for the alarm and SNS topic names."
  type        = string
}

variable "environment" {
  description = "Environment label used in the log group name and the CloudWatch Environment dimension (\"dev\" or \"prod\")."
  type        = string
}

variable "log_group_name" {
  description = "Log group the server container ships to. Empty (default) uses /e2eetext/<environment>/server."
  type        = string
  default     = ""
}

variable "retention_in_days" {
  description = "How long server logs are kept. Short on purpose: the app is end-to-end encrypted and logs should not outlive their usefulness. Must be a value CloudWatch Logs accepts."
  type        = number
  default     = 30
}

variable "kms_key_id" {
  description = "Optional KMS key ARN to encrypt the log group. Empty uses the CloudWatch Logs default encryption. The key policy must allow the logs service."
  type        = string
  default     = ""
}

variable "metrics_namespace" {
  description = "CloudWatch namespace for the error-count metric the alarm watches (same namespace the app metrics use)."
  type        = string
  default     = "E2EEText/Messenger"
}

variable "alarm_enabled" {
  description = "Create the metric filter on error-level log lines, the alarm and the SNS topic (#94)."
  type        = bool
  default     = true
}

variable "alarm_threshold" {
  description = "The alarm fires when this many error-level lines occur within one evaluation period."
  type        = number
  default     = 1
}

variable "alarm_period_seconds" {
  description = "Length of one alarm evaluation period in seconds (multiple of 60)."
  type        = number
  default     = 300
}

variable "alarm_email" {
  description = "Optional email address subscribed to the alarm topic. AWS sends a confirmation message that must be accepted before alerts arrive. Empty creates the topic with no subscriber (subscribe out of band)."
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags applied to the log group, alarm and SNS topic."
  type        = map(string)
  default     = {}
}
