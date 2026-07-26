variable "aws_region" {
  description = "AWS region for the ECR repositories (must match maintenance/ecr/.env's AWS_REGION)."
  type        = string
  default     = "us-east-1"
}

variable "github_org" {
  description = "GitHub organization/user that owns the repository."
  type        = string
  default     = "HrrrXppp"
}

variable "github_repo" {
  description = "GitHub repository name."
  type        = string
  default     = "e2eetext"
}

variable "github_actions_role_name" {
  description = "Name of the existing IAM role that build-images.yml assumes (secrets.AWS_ROLE_ARN). Must match exactly for a zero-diff import."
  type        = string
  default     = "github-actions-ecr-push"
}

variable "github_actions_policy_name" {
  description = "Name of the existing IAM policy granting ECR push access to the role above. Must match exactly for a zero-diff import."
  type        = string
  default     = "github-actions-ecr-push"
}
