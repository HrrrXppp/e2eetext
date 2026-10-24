variable "role_name" {
  description = "IAM role name assumed by terraform.yml's plan job (AWS_TERRAFORM_PLAN_ROLE_ARN). Must match the live role for import."
  type        = string
  default     = "terraform-plan-role"
}

variable "inline_policy_name" {
  description = "Name of the inline role policy (aws_iam_role_policy). Live role uses an inline policy, not a managed policy attachment."
  type        = string
  default     = "terraform-plan-rolePolicy"
}

variable "github_org" {
  description = "GitHub organization/user that owns the repository."
  type        = string
}

variable "github_repo" {
  description = "GitHub repository name."
  type        = string
}

variable "oidc_provider_arn" {
  description = "ARN of the existing GitHub Actions OIDC provider (module.github_oidc output)."
  type        = string
}

variable "state_bucket" {
  description = "S3 bucket holding Terraform state (read-only for this role)."
  type        = string
  default     = "e2eetext-terraform-state"
}

variable "ecr_repository_arns" {
  description = "ECR repository ARNs the plan role may describe."
  type        = list(string)
}

variable "tags" {
  description = "Tags for the IAM role. Default empty to match the untagged live role."
  type        = map(string)
  default     = {}
}

variable "static_site_bucket_names" {
  description = "Names of the static site S3 buckets (#28) the plan role may read for refresh. Empty (default) adds no static-site read permissions."
  type        = list(string)
  default     = []
}
