provider "aws" {
    profile = "parler"
    region = "us-east-1"
}

resource "aws_instance" "parler" {
  count = 22
  ami = "ami-0be2609ba883822ec"
  instance_type = "t3.medium"
  key_name = "default"
  subnet_id = "subnet-8e999ad5"
  vpc_security_group_ids = ["sg-6143db1e"]
}

output "ips" {
    value = aws_instance.parler.*.public_ip
}