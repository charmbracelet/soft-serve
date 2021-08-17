terraform {
  backend "s3" {
    bucket = "charm-terraform-backend"
    key    = "smoothie-development"
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
  default = "ghcr.io/charmbracelet/smoothie-internal:snapshot"
}

variable "force_new_deployment" {
  default = false
}

variable "authorization_keys" {
  default = ""
}

module "smoothie" {
  /* source = "../terraform-aws-smoothie" */
  source  = "app.terraform.io/charm/smoothie/aws"
  version = "0.2.1"

  environment                  = var.environment
  aws_region                   = var.aws_region
  ecs_task_execution_role_name = "smoothieEcsTaskExecutionRole-${var.environment}"
  app_image                    = var.app_image
  app_count                    = 2
  app_ssh_port                 = 23231
  fargate_cpu                  = "1024"
  fargate_memory               = "2048"
  force_new_deployment         = var.force_new_deployment
  app_use_default_ssh_port     = true
  authorization_keys           = var.authorization_keys
}
