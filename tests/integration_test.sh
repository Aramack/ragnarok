#!/bin/bash
#Ensure that the healthcheck responsed with an http 200
echo "Testing healthcheck:"
curl -X HEAD \
  "localhost:2626/healthcheck" \
  -vfs14

if [ $? -ne 0 ]; then
  echo "  Healthcheck failed..."
else
  echo "  Healthcheck succeeded..."
fi

#Ensure that the healthcheck actions work
echo "Testing healthcheck actions (down):"
curl -X POST \
  "localhost:2626/healthcheck/down" \
  -vfs14
if [ $? -ne 0 ]; then
  echo "  healthcheck down succeeded..."
else
  echo "  healthcheck down failed..."
fi

echo "Testing healthcheck actions (up):"
curl -X POST \
  "localhost:2626/healthcheck/up" \
  -vfs14
if [ $? -ne 0 ]; then
  echo "  healthcheck up failed..."
else
  echo "  healthcheck up succeeded..."
fi

#Ensure that we can create sample traffic
echo "Testing traffic creating:"
curl -X POST \
  "localhost:2626/api/traffic" \
  -H "Content-Type: application/json" \
  -d '{
       "threadcount":3,
       "url":
         [
           "https://google.com",
           "https://example.com",
           "https://msn.com",
           "https://bing.com",
           "https://twitch.tv",
         ]
      }' \
  -vfs14m 2
if [ $? -ne 0 ]; then
  echo "  Traffic creation failed..."
else
  echo "  Traffic creation success..."
fi

#Esnsure that we can retriece a record
echo "Testing traffic GET:"
curl -X GET \
  "localhost:2626/api/traffic/0" \
  -vfs14
if [ $? -ne 0 ]; then
  echo "  Traffic get failed..."
else
  echo "  Traffic get success..."
fi

