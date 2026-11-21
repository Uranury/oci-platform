#!/bin/sh
# Installed by Ansible (nginx role). certbot runs this after each renewal so
# nginx picks up the new certificate.
exec systemctl reload nginx.service
