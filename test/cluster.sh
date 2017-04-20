#!/bin/bash 

if [ -z "$TESTCONTAINER" ]; then 
    echo "TESTCONTAINER environment variable must be set"
    exit 1 
fi

NAMESPACE=${TESTCONTAINER}-ns
STARTERVOLUME1=${TESTCONTAINER}-vol1
STARTERVOLUME2=${TESTCONTAINER}-vol2
STARTERVOLUME3=${TESTCONTAINER}-vol3
STARTERCONTAINER1=${TESTCONTAINER}-s1
STARTERCONTAINER2=${TESTCONTAINER}-s2
STARTERCONTAINER3=${TESTCONTAINER}-s3
CMD=$1
DOCKERARGS=
STARTERARGS=

# Cleanup
docker rm -f -v $(docker ps -a | grep ${TESTCONTAINER} | awk '{print $1}') &> /dev/null
docker volume rm -f ${STARTERVOLUME1} ${STARTERVOLUME2} ${STARTERVOLUME3} &> /dev/null

if [ "$CMD" == "start" ]; then
    if [ -z "$ARANGODB" ]; then 
        echo "ARANGODB environment variable must be set"
        exit 1 
    fi

    # Create volumes
    docker volume create ${STARTERVOLUME1} &> /dev/null
    docker volume create ${STARTERVOLUME2} &> /dev/null
    docker volume create ${STARTERVOLUME3} &> /dev/null

    # Setup args 
    if [ -n "$JWTSECRET" ]; then 
        if [ -z "$TMPDIR" ]; then 
            echo "TMPDIR environment variable must be set"
            exit 1 
        fi
        JWTSECRETFILE="$TMPDIR/$TESTCONTAINER-jwtsecret"
        echo "$JWTSECRET" > ${JWTSECRETFILE}
        DOCKERARGS="$DOCKERARGS -v $JWTSECRETFILE:/jwtsecret:ro"
        STARTERARGS="$STARTERARGS --jwtSecretFile=/jwtsecret"
    fi 
    if [ "$SSL" == "auto" ]; then 
        STARTERARGS="$STARTERARGS --sslAutoKeyFile"
    fi

    # Start network namespace
    docker run -d --name=${NAMESPACE} alpine:3.4 sleep 365d

    # Start starters 
    # arangodb/arangodb-starter 0.6.0 or higher is needed.
    docker run -d --name=${STARTERCONTAINER1} --net=container:${NAMESPACE} \
        -v ${STARTERVOLUME1}:/data -v /var/run/docker.sock:/var/run/docker.sock $DOCKERARGS arangodb/arangodb-starter \
        --dockerContainer=${STARTERCONTAINER1} --masterPort=7000 --ownAddress=127.0.0.1 --docker=${ARANGODB} $STARTERARGS
    docker run -d --name=${STARTERCONTAINER2} --net=container:${NAMESPACE} \
        -v ${STARTERVOLUME2}:/data -v /var/run/docker.sock:/var/run/docker.sock $DOCKERARGS arangodb/arangodb-starter \
        --dockerContainer=${STARTERCONTAINER2} --masterPort=7000 --ownAddress=127.0.0.1 --docker=${ARANGODB} $STARTERARGS --join=127.0.0.1
    docker run -d --name=${STARTERCONTAINER3} --net=container:${NAMESPACE} \
        -v ${STARTERVOLUME3}:/data -v /var/run/docker.sock:/var/run/docker.sock $DOCKERARGS arangodb/arangodb-starter \
        --dockerContainer=${STARTERCONTAINER3} --masterPort=7000 --ownAddress=127.0.0.1 --docker=${ARANGODB} $STARTERARGS --join=127.0.0.1
fi