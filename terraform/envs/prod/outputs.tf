output "alb_dns_name" {
  description = "DNS name of the prod ALB — point your production domain's Route 53 alias/CNAME here."
  value       = module.alb.alb_dns_name
}

output "instance_id" {
  description = "Prod EC2 instance ID."
  value       = module.ec2.instance_id
}
