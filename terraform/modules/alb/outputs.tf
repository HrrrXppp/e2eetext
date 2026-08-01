output "alb_arn" {
  description = "ARN of the Application Load Balancer."
  value       = aws_lb.this.arn
}

output "alb_dns_name" {
  description = "DNS name of the ALB — point your domain's Route 53 alias/CNAME here."
  value       = aws_lb.this.dns_name
}

output "server_target_group_arn" {
  description = "ARN of the server (port 8081) target group."
  value       = aws_lb_target_group.server.arn
}

output "client_target_group_arn" {
  description = "ARN of the client (port 8080) target group."
  value       = aws_lb_target_group.client.arn
}
