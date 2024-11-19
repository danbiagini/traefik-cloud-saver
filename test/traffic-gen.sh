#!/bin/bash

while true; do
  curl -H "Host: traefik-test" http://localhost
  sleep 30
done
