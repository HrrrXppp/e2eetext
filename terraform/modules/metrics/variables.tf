variable "name" {
  description = "Name prefix for the resources (e.g. \"e2eetext-dev\"). The Lambda is named \"<name>-metrics\"."
  type        = string
}

variable "environment" {
  description = "Value of the CloudWatch Environment dimension (e.g. \"dev\" or \"prod\")."
  type        = string
}

variable "namespace" {
  description = "CloudWatch namespace the Lambda publishes to. Must match the namespace the server nodes publish WebSocketConnections to (METRICS_CLOUDWATCH_NAMESPACE)."
  type        = string
  default     = "E2EEText/Messenger"
}

variable "lambda_zip_path" {
  description = "Path to the Lambda deployment package built by lambdas/metrics/build.sh (lambdas/metrics/build/metrics.zip)."
  type        = string
}

variable "database_url_secret_arn" {
  description = "ARN of the Secrets Manager secret whose string value is the PostgreSQL connection string (same format as the server's DATABASE_URL). The secret itself is created out of band so the password never enters Terraform state."
  type        = string

  validation {
    condition     = can(regex("^arn:aws[a-z0-9-]*:secretsmanager:[a-z0-9-]+:[0-9]{12}:secret:.+", var.database_url_secret_arn))
    error_message = "database_url_secret_arn must be a Secrets Manager secret ARN. An empty value would be written into the Lambda IAM policy as Resource \"\"."
  }
}

variable "vpc_id" {
  description = "VPC the Lambda runs in (must be able to reach the RDS instance)."
  type        = string
}

variable "subnet_ids" {
  description = "Subnets for the Lambda and the VPC endpoints. Use at most one subnet per Availability Zone (interface endpoints allow only one per AZ)."
  type        = list(string)
}

variable "rds_security_group_id" {
  description = "Security group of the RDS instance. When set, an ingress rule for 5432 from the Lambda's security group is added to it. Leave empty if RDS already allows the Lambda's security group by other means."
  type        = string
  default     = ""
}

variable "create_vpc_endpoints" {
  description = "Create interface VPC endpoints for CloudWatch (monitoring) and Secrets Manager so the Lambda needs no internet/NAT egress. Set false if the VPC already has them or has a NAT gateway."
  type        = bool
  default     = true
}

variable "schedule_expression" {
  description = "EventBridge schedule. 1 minute matches the server nodes' publish interval."
  type        = string
  default     = "rate(1 minute)"
}

variable "log_retention_days" {
  description = "CloudWatch Logs retention for the Lambda log group."
  type        = number
  default     = 30
}

variable "tags" {
  description = "Tags applied to all resources."
  type        = map(string)
  default     = {}
}
