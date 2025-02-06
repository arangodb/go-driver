#!/bin/bash 

IAT=$(date -u +%s)
EXP=$(($IAT+36000))

HEADER='{"alg":"HS256","typ":"JWT"}'
PAYLOAD1='{"exp": '
PAYLOAD2=', "iat":'
PAYLOAD3=', "iss":"arangodb","server_id":"arangodb"}'
PAYLOAD="$PAYLOAD1$EXP$PAYLOAD2$IAT$PAYLOAD3"

PAYLOAD='{"iss":"arangodb","server_id":"arangodb"}'

jwt_header=$(echo $(echo -n '{"alg":"HS256","typ":"JWT"}' | base64) | sed 's/ /_/g' | sed 's/+/-/g' | sed -E s/=+$//)
payload=$(echo $(echo -n "${PAYLOAD}" | base64) | sed 's/ /_/g' | sed 's/+/-/g' |  sed -E s/=+$//)

hexsecret=$(echo -n "$JWTSECRET" | xxd -p | paste -sd "")
hmac_signature=$(echo $(echo -n "${jwt_header}.${payload}" |  openssl dgst -sha256 -mac HMAC -macopt hexkey:$hexsecret -binary | base64 ) | sed 's/\//_/g' | sed 's/+/-/g' | sed -E s/=+$//)

# Create the full token
jwt="${jwt_header}.${payload}.${hmac_signature}"
echo $jwt