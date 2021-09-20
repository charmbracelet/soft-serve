terraform {
  backend "s3" {
    bucket = "charm-terraform-backend"
    key    = "soft-serve-development"
    region = "us-east-1"
  }
}

variable "environment" {
  default = "development"
}

variable "aws_region" {
  default = "us-east-1"
}

variable "app_image" {
  default = "ghcr.io/charmbracelet/soft-serve-internal:snapshot"
}

variable "force_new_deployment" {
  default = false
}

variable "authorization_keys" {
  default = ""
}

module "soft_serve" {
  # source = "../terraform-aws-soft-serve"
  source  = "app.terraform.io/charm/soft-serve/aws"
  version = "0.3.2"

  environment                  = var.environment
  aws_region                   = var.aws_region
  ecs_task_execution_role_name = "softServeEcsTaskExecutionRole-${var.environment}"
  app_image                    = var.app_image
  app_count                    = 2
  app_ssh_port                 = 23231
  fargate_cpu                  = "1024"
  fargate_memory               = "2048"
  force_new_deployment         = var.force_new_deployment
  app_use_default_ssh_port     = true
  authorization_keys           = var.authorization_keys
}
