#!/bin/sh

printf "stage:boot\n"
stprov remote static -A --ip=10.0.2.15/24 --full-host=example.org --url=https://example.org/ospkg.json

printf "stage:network\n"
printf "\n" | stprov remote run -p 2009 --allow=0.0.0.0/0 --otp=sikritpassword
printf "\n"

printf "stage:shutdown\n"
shutdown
