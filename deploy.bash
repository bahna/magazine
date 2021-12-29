#!/bin/bash

# on remote host
USER=root
HOST=bahna.ngo
PROJECT_DIR=/deploy/magazine
REMOTE_ASSETS_DIR=$PROJECT_DIR/assets
REMOTE_SERVICES_DIR=/etc/systemd/system
# local
ASSETS_DIR=build/assets
BINARY_PATH=build/magazine-server
# on both
SERVICE_FILE=magazine.service


echo "* deploying bahna.land"
echo "----------------------"

echo "* stopping the service"
ssh -p 2200 -o "StrictHostKeyChecking no" $USER@$HOST sudo systemctl stop $SERVICE_FILE

echo "* creating directories"
ssh -p 2200 -o "StrictHostKeyChecking no" $USER@$HOST mkdir -p $REMOTE_ASSETS_DIR

echo "* copying the binary"
scp -P 2200 -o "StrictHostKeyChecking no" -q $BINARY_PATH $USER@$HOST:$PROJECT_DIR

echo "* copying assets"
rsync -avz --timeout 10 --delete -e 'ssh -p 2200 -o "StrictHostKeyChecking no"' $ASSETS_DIR/i18n $USER@$HOST:$REMOTE_ASSETS_DIR
rsync -avz --timeout 10 --delete -e 'ssh -p 2200 -o "StrictHostKeyChecking no"' $ASSETS_DIR/static $USER@$HOST:$REMOTE_ASSETS_DIR
rsync -avz --timeout 10 --delete -e 'ssh -p 2200 -o "StrictHostKeyChecking no"' $ASSETS_DIR/templates $USER@$HOST:$REMOTE_ASSETS_DIR

echo "* copying the service file"
scp -P 2200 -o "StrictHostKeyChecking no" -q $SERVICE_FILE $USER@$HOST:$REMOTE_SERVICES_DIR

echo "* updating and restarting the service"
ssh -p 2200 -o "StrictHostKeyChecking no" $USER@$HOST sudo systemctl daemon-reload
ssh -p 2200 -o "StrictHostKeyChecking no" $USER@$HOST sudo systemctl restart $SERVICE_FILE

echo "* done"
