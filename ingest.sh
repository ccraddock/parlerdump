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
  54.175.86.159
  18.234.115.53
  34.235.161.204
  54.162.147.216
  54.208.54.79
  52.54.178.193
  54.196.108.36
  54.172.237.14
  3.90.153.163
  18.232.132.211
  18.215.240.208
  3.90.237.111
  3.85.78.167
  54.196.65.251
  54.172.139.236
  54.88.211.101
  107.21.90.130
  3.85.88.184
  3.89.70.157
  54.166.243.70
  52.71.255.70
  54.209.99.120
)

#
#for server in ${fleetThree[*]}; do
#  echo "setting ip ${server}"
#  ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ec2-user@${server} "sudo yum -y install golang && git clone https://github.com/tkellen/parlerdump.git && mkdir ~/.aws" &
#done
#wait
#for server in ${fleetThree[*]}; do
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
export PARLER_CONCURRENCY=10
cd /home/ec2-user/parlerdump
git pull
wget -q -O - ${url} | go run main.go
EOF
)" &
  sleep 1
done

wait