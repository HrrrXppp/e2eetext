variable "aws_region" {
  description = "AWS region for the ECR repositories (must match maintenance/ecr/.env's AWS_REGION)."
  type        = string
  default     = "us-east-2"
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

variable "oidc_thumbprint_list" {
  description = <<-EOT
    TLS certificate thumbprints for token.actions.githubusercontent.com, passed
    through to module.github_oidc. The default is GitHub's current root CA
    (DigiCert Global Root G2) thumbprint. Only override this if the live,
    manually-created OIDC provider was recorded with a different thumbprint —
    check `aws iam list-open-id-connect-providers` / `get-open-id-connect-provider`
    if the post-import `terraform plan` shows a diff on this field.
  EOT
  type        = list(string)
  default     = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
}
