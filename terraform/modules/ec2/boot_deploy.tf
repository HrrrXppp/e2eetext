# Boot-time deploy: on every boot, ECR login + compose pull (no-op if the
# tag is already local) + compose up. Applied to a live instance via SSM
# (user_data alone cannot reconfigure brownfield dig — it only runs on
# first launch, and changing it is ForceNew). New instances still get the
# same unit via user_data when manage_boot_deploy is true.

locals {
  boot_deploy_enabled = var.manage_boot_deploy
  boot_deploy_name    = var.name != "" ? "${var.name}-boot-deploy" : "e2eetext-boot-deploy"
  boot_deploy_script = local.boot_deploy_enabled ? templatefile("${path.module}/templates/install-boot-deploy.sh.tftpl", {
    repo_root = var.boot_deploy_repo_root
    image_tag = var.boot_deploy_image_tag
    start_now = var.boot_deploy_start_now ? "true" : "false"
  }) : ""
  # Separate template + base64 avoids nested-heredoc collisions with the
  # install script body (and keeps the ternary parseable).
  boot_deploy_user_data = local.boot_deploy_enabled ? templatefile("${path.module}/templates/user-data-boot-deploy.sh.tftpl", {
    script_b64 = base64encode(local.boot_deploy_script)
  }) : null
}

check "boot_deploy_requires_instance_profile" {
  assert {
    condition     = !var.manage_boot_deploy || var.create_iam_instance_profile
    error_message = "manage_boot_deploy=true requires create_iam_instance_profile=true (ECR pull + SSM agent IAM)."
  }
}

check "boot_deploy_repo_root_absolute" {
  assert {
    condition     = !var.manage_boot_deploy || startswith(var.boot_deploy_repo_root, "/")
    error_message = "boot_deploy_repo_root must be an absolute path on the instance (e.g. /home/ec2-user/e2eetext)."
  }
}

check "boot_deploy_image_tag_set" {
  assert {
    condition     = !var.manage_boot_deploy || var.boot_deploy_image_tag != ""
    error_message = "manage_boot_deploy=true requires boot_deploy_image_tag (pin it in envs/*/ec2_image_tag.tf)."
  }
}

resource "aws_iam_role_policy_attachment" "ssm_managed_instance_core" {
  count = var.create_iam_instance_profile && var.manage_boot_deploy ? 1 : 0

  role       = aws_iam_role.this[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_ssm_document" "boot_deploy" {
  count = local.boot_deploy_enabled ? 1 : 0

  name            = local.boot_deploy_name
  document_type   = "Command"
  document_format = "JSON"

  content = jsonencode({
    schemaVersion = "2.2"
    description   = "Install e2eetext systemd boot unit (ECR pull + compose up)"
    mainSteps = [
      {
        action = "aws:runShellScript"
        name   = "InstallBootDeploy"
        inputs = {
          timeoutSeconds = "900"
          runCommand     = [local.boot_deploy_script]
        }
      }
    ]
  })

  tags = var.tags
}

resource "aws_ssm_association" "boot_deploy" {
  count = local.boot_deploy_enabled ? 1 : 0

  name             = aws_ssm_document.boot_deploy[0].name
  association_name = local.boot_deploy_name

  targets {
    key    = "InstanceIds"
    values = [aws_instance.this.id]
  }

  # Re-apply when the document changes (e.g. IMAGE_TAG pin) without waiting
  # for a cron window.
  apply_only_at_cron_interval = false
}
