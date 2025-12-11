resource "local_file" "encrypted_config" {
  content = templatefile("${path.module}/encryption-provider-config.tpl", {
    encryption_secret_key = var.encryption_secret_key
  })
  filename = "${path.module}/encryption-provider-config.yaml"
}

resource "aws_instance" "rke2_node" {
  ami                         = var.ami_id
  instance_type               = var.instance_type
  key_name                    = var.key_name
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [var.security_group_id]
  associate_public_ip_address = var.preserve_eip ? false : true

  root_block_device {
    volume_size           = var.root_volume_size
    delete_on_termination = true
    volume_type           = "gp2"
  }

  tags = {
    Name = "${var.prefix}-rke2-node"
  }
}

resource "aws_eip" "static_ip" {
  count  = var.preserve_eip ? 1 : 0
  domain = "vpc"

  tags = {
    Name = "${var.prefix}-static-ip"
  }
}

resource "aws_eip_association" "eip_association" {
  count         = var.preserve_eip ? 1 : 0
  instance_id   = aws_instance.rke2_node.id
  allocation_id = aws_eip.static_ip[0].id
}

locals {
  rke2_host_ip = var.preserve_eip ? aws_eip.static_ip[0].public_ip : aws_instance.rke2_node.public_ip
}

resource "null_resource" "provision_rke2" {
  depends_on = [aws_instance.rke2_node]

  # Transfer install_rke2.sh
  provisioner "file" {
    source      = "${path.module}/install_rke2.sh"
    destination = "/home/ubuntu/install_rke2.sh"

    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(var.private_key_path)
      host        = local.rke2_host_ip
    }
  }

  # Transfer install_rancher.sh
  provisioner "file" {
    source      = "${path.module}/install_rancher.sh"
    destination = "/home/ubuntu/install_rancher.sh"

    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(var.private_key_path)
      host        = local.rke2_host_ip
    }
  }

  provisioner "file" {
    source      = local_file.encrypted_config.filename
    destination = "/home/ubuntu/encryption-provider-config.yaml"

    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(var.private_key_path)
      host        = local.rke2_host_ip
    }
  }

  # Run scripts sequentially
  provisioner "remote-exec" {
    inline = [
      "chmod +x /home/ubuntu/install_rke2.sh /home/ubuntu/install_rancher.sh",
      "sudo -i bash /home/ubuntu/install_rke2.sh '${var.rke2_version}' '${var.cert_manager_version}' '${var.rancher_repo_url}'",
      "sudo -i bash /home/ubuntu/install_rancher.sh '${var.rancher_version}' '${var.rancher_password}' '${var.rancher_repo_url}' '${var.install_rancher}'"
    ]

    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(var.private_key_path)
      host        = local.rke2_host_ip
    }
  }
}

resource "null_resource" "copy_kubeconfig" {
  depends_on = [null_resource.provision_rke2]

  provisioner "local-exec" {
    command = "scp -i ${var.private_key_path} -o StrictHostKeyChecking=no ubuntu@${local.rke2_host_ip}:/tmp/rke2.yaml ./rke2-kubeconfig.yaml"
  }
}

resource "null_resource" "move_kubeconfig_local" {
  depends_on = [null_resource.copy_kubeconfig]

  provisioner "local-exec" {
    command = <<EOT
      mkdir -p ~/.kube
      mv ./rke2-kubeconfig.yaml ~/.kube/config
      chmod 600 ~/.kube/config
      echo "✅ kubeconfig placed at ~/.kube/config"
    EOT
  }
}

resource "null_resource" "update_yaml" {
  triggers = {
    subnet_id = var.subnet_id
    vpc_id    = var.vpc_id
    host_ip   = local.rke2_host_ip
  }

  provisioner "local-exec" {
    # The command is now clean and contains no sensitive data
    command = "/bin/bash ${path.module}/update_configs.sh"

    # Pass secrets securely as environment variables
    environment = {
      INPUT_CONFIG_PATH  = var.input_cluster_config
      SUBNET_ID          = var.subnet_id
      VPC_ID             = var.vpc_id
      CATTLE_CONFIG_PATH = var.cattle_config
      RKE2_HOST_IP       = local.rke2_host_ip
    }
  }
}
