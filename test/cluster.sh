#!/bin/bash 

if [ -z "$TESTCONTAINER" ]; then 
    echo "TESTCONTAINER environment variable must be set"
    exit 1 
fi

if [ -z "$STARTER" ]; then
    echo "STARTER environment variable must be set"
    exit 1
fi

NAMESPACE=${TESTCONTAINER}-ns
STARTERVOLUME=${TESTCONTAINER}-vol
STARTERCONTAINER=${TESTCONTAINER}-s
CMD=$1
DOCKERARGS=
STARTERARGS=
# Cleanup
docker rm -f -v $(docker ps -a | grep ${TESTCONTAINER} | awk '{print $1}') &> /dev/null
docker volume rm -f ${STARTERVOLUME} &> /dev/null
if [ "$CMD" == "start" ]; then
    if [ -z "$ARANGODB" ]; then 
        echo "ARANGODB environment variable must be set"
        exit 1 
    fi

    # Create volumes
    docker volume create ${STARTERVOLUME} &> /dev/null

    # Setup args
    if [ -n "$JWTSECRET" ]; then
        if [ -z "$TMPDIR" ]; then 
            echo "TMPDIR environment variable must be set"
            exit 1 
        fi
        JWTSECRETFILE="$TMPDIR/$TESTCONTAINER-jwtsecret"
        echo "$JWTSECRET" > ${JWTSECRETFILE}
        DOCKERARGS="$DOCKERARGS -v $JWTSECRETFILE:/jwtsecret:ro"
        STARTERARGS="$STARTERARGS --auth.jwt-secret=/jwtsecret"
    fi 
    if [ "$SSL" == "auto" ]; then 
        STARTERARGS="$STARTERARGS --ssl.auto-key"
    fi
    if [ -n "$ARANGO_LICENSE_KEY" ]; then
        DOCKERARGS="$DOCKERARGS -e ARANGO_LICENSE_KEY=$ARANGO_LICENSE_KEY"
    fi
    if [ -n "$ENABLE_BACKUP" ]; then
        STARTERARGS="$STARTERARGS --all.backup.api-enabled=true"
    fi
    if [ -n "$ENABLE_DATABASE_EXTENDED_NAMES" ]; then
        STARTERARGS="$STARTERARGS --all.database.extended-names-databases=true"
    fi
    if [[ "$OSTYPE" == "darwin"* ]]; then
        DOCKERPLATFORMARG="--platform linux/x86_64"
        DOCKERARGS="$DOCKERARGS $DOCKERPLATFORMARG"
    fi
    STARTERARGS="$STARTERARGS --docker.container=$STARTERCONTAINER"
    #STARTERARGS="$STARTERARGS --docker.container=$STARTERCONTAINER --args.all.cluster.failed-write-concern-status-code=503"

    if [ -z "$STARTERPORT" ]; then
        STARTERPORT=7000
    fi

    if [ -n "$DEBUG_PORT" ]; then
        DOCKER_DEBUG_PORT="-p $DEBUG_PORT:$DEBUG_PORT"
    fi

    set -x

    # Start network namespace
    docker run -d --name=${NAMESPACE} $DOCKERPLATFORMARG $DOCKER_DEBUG_PORT "${ALPINE_IMAGE}" sleep 365d

    echo "Starting starter container: ${ARANGODB}"

    # Start starters 
    # arangodb/arangodb-starter 0.7.0 or higher is needed.
    docker run -d --name=${STARTERCONTAINER} --net=container:${NAMESPACE} \
        -v ${STARTERVOLUME}:/data -v /var/run/docker.sock:/var/run/docker.sock $DOCKERARGS \
        ${STARTER} \
        --starter.port=${STARTERPORT} --starter.address=127.0.0.1 \
        --docker.image=${ARANGODB} \
        --starter.local --starter.mode=${STARTERMODE} --all.log.level=debug --all.log.output=+ --log.verbose $STARTERARGS
fi
