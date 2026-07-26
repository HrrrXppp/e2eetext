variable "repository_names" {
  description = "Names of the ECR repositories to manage. Must match names already created by maintenance/ecr/create-repos.sh for a zero-diff import."
  type        = list(string)
  default     = ["e2eetext-server", "e2eetext-client"]
}

variable "tags" {
  description = "Tags applied to every ECR repository."
  type        = map(string)
  default     = {}
}
