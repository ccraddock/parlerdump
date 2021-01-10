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
)

(
  for url in ${urls[*]}; do
    wget -q -O - "https://donk.sh/06d639b2-0252-4b1e-883b-f275eff7e792/${url}"
  done
) | go run main.go