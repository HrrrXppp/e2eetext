variable "name" {
  description = "Base name for the ALB and its target groups (e.g. \"e2eetext-dev\"). Must match the live ALB's name for a zero-diff import."
  type        = string
}

variable "vpc_id" {
  description = "VPC ID the target groups live in (module.network's output)."
  type        = string
}

variable "subnet_ids" {
  description = "Public subnet IDs for the internet-facing ALB (module.network's output, or an explicit subset)."
  type        = list(string)
}

variable "security_group_ids" {
  description = <<-EOT
    Existing security group ID(s) for the ALB, allowing inbound 80/443 from
    0.0.0.0/0. Not created by this module — create-alb.example.sh treats
    ALB_SG_ID as a pre-existing input, and this module mirrors that. Set to
    the real, already-live security group ID for a zero-diff import.
  EOT
  type        = list(string)
}

variable "instance_id" {
  description = "EC2 instance ID to register in both target groups (module.ec2's output)."
  type        = string
}

variable "server_target_group_name" {
  description = <<-EOT
    Name of the server (port 8081) target group. Leave empty (default) to
    derive "<name>-server". Set explicitly when the live target group's
    name doesn't follow that convention (the live dev TG is literally named
    "dev-server") — target group names are ForceNew, so a mismatch here
    means a destroy/create (downtime) instead of a zero-diff import.
  EOT
  type        = string
  default     = ""
}

variable "client_target_group_name" {
  description = <<-EOT
    Name of the client (port 8080) target group. Leave empty (default) to
    derive "<name>-client". Same ForceNew caveat as
    server_target_group_name (the live dev TG is literally named "client").
  EOT
  type        = string
  default     = ""
}

variable "health_check_healthy_threshold" {
  description = "Consecutive successful checks before a target is healthy. Provider default is 3; the live dev TGs (console-made) use 5."
  type        = number
  default     = 3
}

variable "health_check_unhealthy_threshold" {
  description = "Consecutive failed checks before a target is unhealthy. Provider default is 3; the live dev TGs use 2."
  type        = number
  default     = 3
}

variable "server_health_check_port" {
  description = "Health-check port for the server TG. Default \"traffic-port\"; the live dev TG pins it to \"8081\" explicitly (functionally identical, but a plan diff if it doesn't match state)."
  type        = string
  default     = "traffic-port"
}

variable "client_health_check_port" {
  description = "Health-check port for the client TG. Default \"traffic-port\"; the live dev TG uses \"traffic-port\" (not pinned to 8080)."
  type        = string
  default     = "traffic-port"
}

variable "client_health_check_path" {
  description = "Health-check path for the client TG. Default \"/health\" (script-shaped); live-dev uses \"/\"."
  type        = string
  default     = "/health"
}

variable "idle_timeout" {
  description = <<-EOT
    ALB idle timeout in seconds. Defaults to 3600, matching
    create-alb.example.sh's explicit setting for /ws WebSocket support.
    The live dev ALB, however, runs the AWS default of 60 — pass the real
    live value when importing so plan converges without a diff.
  EOT
  type        = number
  default     = 3600
}

variable "https_default_action" {
  description = <<-EOT
    Shape of the HTTPS listener's default action:
      - "forward-client": forward to the client target group (what
        create-alb.example.sh sets up).
      - "fixed-404": a fixed 404 text response (what the live dev ALB
        actually has — its client routing is expressed as explicit
        path-pattern rules instead of a catch-all default).
  EOT
  type        = string
  default     = "forward-client"

  validation {
    condition     = contains(["forward-client", "fixed-404"], var.https_default_action)
    error_message = "https_default_action must be \"forward-client\" or \"fixed-404\"."
  }
}

variable "fixed_404_message_body" {
  description = "Response body for the \"fixed-404\" default action. Default matches the live dev listener's body byte-for-byte for a zero-diff import."
  type        = string
  default     = "Don't have rule"
}

variable "api_rule_priority" {
  description = "Priority of the /api listener rule. create-alb.example.sh uses 10; the live dev ALB has it at 100 (10-14 are taken by hand-made client rules there)."
  type        = number
  default     = 10
}

variable "api_path_patterns" {
  description = "Path patterns for the /api listener rule. create-alb.example.sh uses [\"/api*\"]; the live dev rule matches [\"/api/*\"]."
  type        = list(string)
  default     = ["/api*"]
}

variable "create_health_rule" {
  description = "Whether to create the /health -> server listener rule (create-alb.example.sh's rule 20). The live dev ALB has no such rule — set false there so import stays zero-diff."
  type        = bool
  default     = true
}

variable "health_rule_priority" {
  description = "Priority of the /health listener rule (when created)."
  type        = number
  default     = 20
}

variable "additional_https_rules" {
  description = <<-EOT
    Extra HTTPS listener rules beyond the built-in /api (and optional
    /health) rules. Each entry needs a unique priority, path pattern(s),
    and target ("client" or "server"). Used on live-dev to manage the
    hand-made SPA routes at priorities 10–14 (/, /assets/*, /oauth/*,
    /chats, /instance.json → client TG). Leave empty for script-shaped
    prod greenfield (default catch-all forward covers SPA routes).
  EOT
  type = list(object({
    priority      = number
    path_patterns = list(string)
    target        = string
  }))
  default = []

  validation {
    condition = alltrue([
      for r in var.additional_https_rules : contains(["client", "server"], r.target)
    ])
    error_message = "additional_https_rules[].target must be \"client\" or \"server\"."
  }
}

variable "acm_certificate_arn" {
  description = "ACM certificate ARN for the HTTPS listener (must be in the same region as the ALB)."
  type        = string
}

variable "tags" {
  description = "Tags applied to the ALB and target groups."
  type        = map(string)
  default     = {}
}
