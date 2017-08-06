#!/bin/bash
#Ensure that the healthcheck responsed with an http 200
echo "Testing healthcheck:"
curl -X HEAD \
  "localhost:2626/healthcheck" \
  -vfs14

if [ $? -ne 0 ]; then
  echo -e "\tHealthcheck failed..." >&2
else
  echo -e "\tHealthcheck succeeded..."
fi

#Ensure that the healthcheck actions work
echo "Testing healthcheck actions (down):"
curl -X POST \
  "localhost:2626/healthcheck/down" \
  -vfs14
if [ $? -ne 0 ]; then
  echo -e "\thealthcheck down succeeded..."
else
  echo -e "\thealthcheck down failed..." >&2
fi

echo "Testing healthcheck actions (up):"
curl -X POST \
  "localhost:2626/healthcheck/up" \
  -vfs14
if [ $? -ne 0 ]; then
  echo -e "\thealthcheck up failed..." >&2
else
  echo -e "\thealthcheck up succeeded..."
fi

#Ensure that we can create sample traffic
echo "Testing traffic creating:"
curl -X POST \
  "localhost:2626/api/traffic" \
  -H "Content-Type: application/json" \
  -d '{
        "headers":
          {
            "cookie":"server=Staging1",
            "User-Agent":"Go-http-client/1.1-infratraffic"
          },
        "iteration":2,
        "threadcount":3,
        "url":
          [
           "https://google.com",
           "https://example.com",
           "https://msn.com",
           "https://bing.com",
           "https://twitch.tv"
          ]
      }' \
  -vfs14m 2
if [ $? -ne 0 ]; then
  echo -e "\tTraffic creation failed..." >&2
else
  echo -e "\tTraffic creation success..."
fi

#Esnsure that we can retriece a record
echo "Testing traffic GET:"
curl -X GET \
  "localhost:2626/api/traffic/0" \
  -vfs14
if [ $? -ne 0 ]; then
  echo -e "\tTraffic get failed..." >&2
else
  echo -e "\tTraffic get success..."
fi

