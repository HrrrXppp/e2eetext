variable "github_org" {
  description = "GitHub organization/user that owns the repository (e.g. HrrrXppp)."
  type        = string
}

variable "github_repo" {
  description = "GitHub repository name (e.g. e2eetext)."
  type        = string
}

variable "oidc_thumbprint_list" {
  description = <<-EOT
    TLS certificate thumbprints for token.actions.githubusercontent.com's certificate
    authority chain. GitHub's current root CA (DigiCert Global Root G2) thumbprint is
    listed by default. Only relevant on initial creation; when importing an existing
    provider this value must match what AWS already has recorded, or Terraform will
    show a diff. AWS also accepts an empty list and verifies the chain itself for
    GitHub/GitLab's well-known OIDC issuers on recent provider versions.
  EOT
  type        = list(string)
  default     = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
}

variable "role_name" {
  description = "Name of the IAM role assumed by GitHub Actions via OIDC. Must match the role already created manually, for a zero-diff import."
  type        = string
  default     = "github-actions-ecr-push"
}

variable "policy_name" {
  description = "Name of the IAM policy granting ECR push access, attached to role_name. Must match the policy already created manually, for a zero-diff import."
  type        = string
  default     = "github-actions-ecr-push"
}

variable "ecr_repository_arns" {
  description = "ARNs of the ECR repositories the role is allowed to push/pull. Scopes the permissions policy to just these repos, matching the manually-created policy."
  type        = list(string)
}

variable "tags" {
  description = "Tags applied to the IAM role."
  type        = map(string)
  default     = {}
}
