variable "ami_id" {}
variable "instance_type" {}
variable "key_name" { sensitive = true }
variable "root_volume_size" {}
variable "prefix" { sensitive = true }
variable "subnet_id" { sensitive = true }
variable "vpc_id" { sensitive = true }
variable "security_group_id" { sensitive = true }
variable "private_key_path" { sensitive = true }
variable "preserve_eip" {
  type    = bool
  default = true
}
variable "rke2_version" {}
variable "cert_manager_version" {}
variable "encryption_secret_key" { sensitive = true }
variable "input_cluster_config" { sensitive = true }
variable "cattle_config" { sensitive = true }
variable "rancher_version" {
}

variable "rancher_password" {
  description = "Bootstrap password for Rancher"
  type        = string
  sensitive   = true
}
variable "rancher_repo_url" {
}

variable "install_rancher" {
}
