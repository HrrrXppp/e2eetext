# Boot-time deploy: on every boot, ECR login + compose pull (no-op if the
# tag is already local) + compose up. Applied to a live instance via SSM
# (user_data alone cannot reconfigure brownfield dig — it only runs on
# first launch, and changing it is ForceNew). New instances still get the
# same unit via user_data when manage_boot_deploy is true.

locals {
  boot_deploy_enabled = var.manage_boot_deploy
  boot_deploy_name    = var.name != "" ? "${var.name}-boot-deploy" : "e2eetext-boot-deploy"
  # Image tags are SSM association parameters, not baked into this
  # script. A parameter change updates the association, and that update
  # runs it. Baking the tag into the document does not: the association
  # resource stays unchanged, so the instance keeps the previous images.
  boot_deploy_script = local.boot_deploy_enabled ? templatefile("${path.module}/templates/install-boot-deploy.sh.tftpl", {
    repo_root   = var.boot_deploy_repo_root
    repo_url    = var.boot_deploy_repo_url
    repo_branch = var.boot_deploy_repo_branch
    start_now   = var.boot_deploy_start_now ? "true" : "false"
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
    error_message = "manage_boot_deploy=true requires boot_deploy_image_tag (pin it in envs/*/version.tf)."
  }
}

check "boot_deploy_image_tag_safe" {
  assert {
    condition     = !var.manage_boot_deploy || can(regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$", var.boot_deploy_image_tag))
    error_message = "boot_deploy_image_tag must be an ECR tag (letters, digits, dot, underscore, hyphen)."
  }
}

check "boot_deploy_repo_url_set" {
  assert {
    condition     = !var.manage_boot_deploy || var.boot_deploy_repo_url != ""
    error_message = "manage_boot_deploy=true requires boot_deploy_repo_url (public clone URL; do not embed a token)."
  }
}

resource "aws_iam_role_policy_attachment" "ssm_managed_instance_core" {
  count = var.create_iam_instance_profile && var.manage_boot_deploy ? 1 : 0

  role       = aws_iam_role.this[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

# Replacing the association (instead of updating it) is what makes apply
# wait for the rollout. An in-place update starts a run from the previous
# association version, which has no parameters, against the new document,
# which requires ImageTag. That run fails immediately and leaves the
# containers untouched. The parameter default covers that race; the
# replacement is the run apply waits for.
resource "terraform_data" "boot_deploy_rollout" {
  count = local.boot_deploy_enabled ? 1 : 0

  input = {
    image_tag  = var.boot_deploy_image_tag
    script_sha = sha256(local.boot_deploy_script)
  }
}

resource "aws_ssm_document" "boot_deploy" {
  count = local.boot_deploy_enabled ? 1 : 0

  name            = local.boot_deploy_name
  document_type   = "Command"
  document_format = "JSON"

  content = jsonencode({
    schemaVersion = "2.2"
    description   = "Install Docker/git if needed, clone the repo if missing, install the e2eetext systemd boot unit (ECR pull + compose up)"
    parameters = {
      ImageTag = {
        type           = "String"
        description    = "ECR tag for the e2eetext-server container"
        default        = var.boot_deploy_image_tag
        allowedPattern = "^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$"
      }
    }
    mainSteps = [
      {
        action = "aws:runShellScript"
        name   = "InstallBootDeploy"
        inputs = {
          timeoutSeconds = "1800"
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
  # Pin the version this apply just published. Without it, an association
  # created against an older document keeps running that older document.
  document_version = tostring(aws_ssm_document.boot_deploy[0].latest_version)

  parameters = {
    ImageTag = var.boot_deploy_image_tag
  }

  targets {
    key    = "InstanceIds"
    values = [aws_instance.this.id]
  }

  # An association update runs immediately. Changing ImageTag is an
  # association update, so apply pulls that tag instead of waiting for
  # the next boot. replace_triggered_by makes a tag or script change a
  # new association, and Create is what honors the wait below.
  apply_only_at_cron_interval      = false
  wait_for_success_timeout_seconds = 900

  lifecycle {
    replace_triggered_by = [terraform_data.boot_deploy_rollout[0]]
  }
}
