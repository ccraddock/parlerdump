#!/usr/bin/env bash
urls=(
  VID000.txt
  VID001.txt
  VID002.txt
  VID003.txt
  VID004.txt
  VID005.txt
  VID006.txt
  VID007.txt
  VID008.txt
  VID009.txt
  VID010.txt
  VID011.txt
  VID012.txt
  VID013.txt
  VID014.txt
  VID015.txt
  VID016.txt
  VID017.txt
  VID018.txt
  VID019.txt
  VID020.txt
  VID021.txt  
)

servers=(
  18.212.128.192
  3.80.199.56
  18.232.156.3
  54.227.168.214
  54.164.239.32
  54.81.240.203
  54.80.122.1
  3.94.130.12
  54.81.200.140
  34.229.209.41
  54.196.97.208
  54.237.208.171
  54.225.15.134
  54.174.195.14
  18.232.79.19
  54.198.251.162
  184.73.92.120
  3.89.184.145
  3.84.207.90
  54.237.247.223
  54.166.214.78
  18.234.30.176
)

#for server in ${servers[*]}; do
#  echo "setting ip ${server}"
#  ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ec2-user@${server} "sudo amazon-linux-extras install epel -y && sudo yum install -y perl-Image-ExifTool.noarch && sudo yum -y install golang exiftool && git clone https://github.com/tkellen/parlerdump.git && mkdir ~/.aws" &
#done
#wait
#for server in ${servers[*]}; do
#  scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ~/.aws/credentials ec2-user@${server}:~/.aws &
#  scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ~/.aws/config ec2-user@${server}:~/.aws &
#done
#wait

for i in ${!urls[@]}; do
  url="https://donk.sh/06d639b2-0252-4b1e-883b-f275eff7e792/${urls[$i]}"
  server="${servers[$i]}"
  echo "starting ${server}"
  ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ec2-user@${server} "$(cat <<EOF
export AWS_DEFAULT_REGION=us-east-1
export AWS_PROFILE=parler
export PARLER_BUCKET=parlerdump
export PARLER_CONCURRENCY=20
cd /home/ec2-user/parlerdump
git pull
wget -q -O - ${url} | go run meta.go
EOF
)" &
  sleep 1
done

wait