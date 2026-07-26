# Mirrors maintenance/ecr/create-repos.sh exactly:
#   aws ecr create-repository \
#     --repository-name "$repo" \
#     --image-scanning-configuration scanOnPush=true \
#     --encryption-configuration encryptionType=AES256
resource "aws_ecr_repository" "this" {
  for_each = toset(var.repository_names)

  name                 = each.value
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = var.tags
}
