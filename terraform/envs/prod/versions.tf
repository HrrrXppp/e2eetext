terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Same S3 bucket as envs/shared, separate state key. Native S3 state
  # locking (use_lockfile) needs Terraform >= 1.10 and no DynamoDB table.
  backend "s3" {
    bucket       = "e2eetext-terraform-state"
    key          = "envs/prod/terraform.tfstate"
    region       = "us-east-2"
    use_lockfile = true
    encrypt      = true
  }
}
