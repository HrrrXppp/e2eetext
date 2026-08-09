terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.34.0, < 7.0.0"
    }
  }

  # One-time manual bootstrap required before `terraform init` will work:
  # create the "e2eetext-terraform-state" S3 bucket (versioning enabled) in
  # this account/region. Native S3 state locking (use_lockfile) needs
  # Terraform >= 1.10 and no separate DynamoDB lock table.
  backend "s3" {
    bucket       = "e2eetext-terraform-state"
    key          = "envs/shared/terraform.tfstate"
    region       = "us-east-2"
    use_lockfile = true
    encrypt      = true
  }
}
