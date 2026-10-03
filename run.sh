#!/bin/sh
# builds the two images (backend + frontend) and starts the containers.
# then open http://localhost in the browser
docker compose up --build -d
docker ps -a --size --filter "name=social-"
