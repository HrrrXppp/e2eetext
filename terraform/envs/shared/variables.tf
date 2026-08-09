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
  # The live role is actually named "github-actions-ecr-role", not
  # "github-actions-ecr-push" — confirmed via the PR #29 CI plan, which
  # showed the role "must be replaced" (destroy + create under the wrong
  # name) instead of matching the imported live resource.
  default = "github-actions-ecr-role"
}

variable "github_actions_inline_policy_name" {
  description = "Inline policy name on the GitHub Actions ECR role (live: github-actions-ecr-rolePolicy). Not a managed policy."
  type        = string
  default     = "github-actions-ecr-rolePolicy"
}

variable "oidc_thumbprint_list" {
  description = <<-EOT
    TLS certificate thumbprints for token.actions.githubusercontent.com, passed
    through to module.github_oidc. The default below matches the live,
    manually-created OIDC provider's actual recorded thumbprint (confirmed via
    `aws iam get-open-id-connect-provider`) — it is NOT the commonly-copied
    "well-known" GitHub thumbprint seen in older tutorials/docs
    (6938fd4d98bab03faadb97b34396831e3780aea1), which does not match what's
    live here. This value can rotate if GitHub changes its certificate chain's
    CA, so re-verify against AWS if a future `terraform plan` shows a diff on
    this field.
  EOT
  type        = list(string)
  default     = ["ab9d0263244dd0326eb67015705a667e79cfe998"]
}

variable "terraform_plan_role_name" {
  description = "Name of the IAM role terraform.yml assumes for plan (AWS_TERRAFORM_PLAN_ROLE_ARN)."
  type        = string
  default     = "terraform-plan-role"
}

variable "terraform_plan_inline_policy_name" {
  description = "Inline policy name on the terraform-plan role (live: terraform-plan-rolePolicy)."
  type        = string
  default     = "terraform-plan-rolePolicy"
}

variable "terraform_state_bucket" {
  description = "S3 bucket for Terraform state; the plan role gets read-only access."
  type        = string
  default     = "e2eetext-terraform-state"
}
