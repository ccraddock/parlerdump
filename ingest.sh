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
  52.91.69.112
  34.207.169.118
  3.92.143.42
  52.87.195.155
  184.73.92.122
  54.80.175.140
  54.91.21.121
  18.234.178.176
  54.158.120.140
  54.89.91.17
  3.89.149.192
  52.90.11.150
  35.153.204.96
  54.86.170.164
  3.84.78.101
  34.228.227.136
  35.171.151.203
  18.215.232.28
  3.84.177.242
  54.84.157.9
  54.161.128.88
  54.159.74.199
)

for server in ${servers[*]}; do
  echo "setting ip ${server}"
  ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ec2-user@${server} "sudo yum -y install golang && git clone https://github.com/tkellen/parlerdump.git && mkdir ~/.aws" &
done
wait
for server in ${servers[*]}; do
  scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ~/.aws/credentials ec2-user@${server}:~/.aws &
  scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ~/.aws/config ec2-user@${server}:~/.aws &
done
wait

for i in ${!urls[@]}; do
  url="https://donk.sh/06d639b2-0252-4b1e-883b-f275eff7e792/${urls[$i]}"
  server="${servers[$i]}"
  echo "starting ${server}"
  ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ec2-user@${server} "$(cat <<EOF
export AWS_DEFAULT_REGION=us-east-1
export AWS_PROFILE=parler
export PARLER_BUCKET=parlerdump
export PARLER_CONCURRENCY=10
cd /home/ec2-user/parlerdump
git pull
wget -q -O - ${url} | tac | go run main.go
EOF
)" &
  sleep 1
done

wait