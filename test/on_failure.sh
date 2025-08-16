#!/bin/bash
exit 1
echo "failure!\n"

echo "\nARANGODB-STARTER logs: "
docker logs ${TESTCONTAINER}-s

echo -n "\nARANGODB-S-* logs:"
docker ps -f name=${TESTCONTAINER}-s- --format "{{.ID}} {{.Names}}" | xargs -L1 bash -c 'echo -e "\n\tLogs from $1:"; docker logs $0'

if [ -n "${DUMP_AGENCY_ON_FAILURE}" ] && [ "${TEST_MODE}" = "cluster" ]; then
    echo "\nAgency dump..."
    
    if [ "${TEST_SSL}" = "auto" ]; then
        PROTOCOL='https'
    else
        PROTOCOL='http'
    fi

    if [ "${TEST_AUTH}" = "jwt" ] || [ "${TEST_AUTH}" = "rootpw" ] || [ "${TEST_AUTH}" = "jwtsuper" ]; then
        JWT_TOKEN=$(bash -c "JWTSECRET=$TEST_JWTSECRET; source ./test/jwt.sh")
        AUTH="-H 'authorization: bearer $JWT_TOKEN'"
    else
        AUTH=""
    fi

    SED_DOCKER_NAME_TO_ENDPOINT="s/.*-([a-zA-Z0-9.-]+)-([0-9]+)$/${PROTOCOL}:\/\/\1:\2/"
    ANY_ENDPOINT=$(docker ps -f name=${TESTCONTAINER}-s-agent --format '{{.Names}}'  | head -n 1 | sed -E $SED_DOCKER_NAME_TO_ENDPOINT)
    echo "Any agent endpoint: $ANY_ENDPOINT"

    # _api/agency/config returns leader endpoint with protocol that is usually not supported by curl 
    AGENCY_CONFIG=$(bash -c "curl -k --no-progress-meter ${AUTH} ${ANY_ENDPOINT}/_api/agency/config")
    
    # same as: jq -r '.configuration.pool[.leaderId]'
    LEADER_ENDPOINT_WITH_UNSUPPORTED_PROTOCOL=$(echo $AGENCY_CONFIG | go run ./test/json_agency_config_parse_leader_id/json_agency_config_parse_leader_id.go | cat)
    SED_UNSUPPORTED_PROTOCOL_ENDPOINT_TO_ENDPOINT="s/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//${PROTOCOL}:\/\//"
    LEADER_ENDPOINT=$(echo $LEADER_ENDPOINT_WITH_UNSUPPORTED_PROTOCOL | sed $SED_UNSUPPORTED_PROTOCOL_ENDPOINT_TO_ENDPOINT)
    
    if expr "$LEADER_ENDPOINT" : "^$PROTOCOL" > /dev/null; then
        echo "Leader agent endpoint: $LEADER_ENDPOINT"
        DUMP_FILE_PATH=$DUMP_AGENCY_ON_FAILURE
        mkdir -p $(dirname ${DUMP_FILE_PATH})
        AGENCY_DUMP=$(bash -c "curl -Lk --no-progress-meter ${AUTH} ${LEADER_ENDPOINT}/_api/agency/state")
        echo $AGENCY_DUMP > $DUMP_FILE_PATH
        echo "Agency dump created at $(realpath $DUMP_FILE_PATH)"
    fi
fi

echo "\nV${MAJOR_VERSION} Tests with ARGS: TEST_MODE=${TEST_MODE} TEST_AUTH=${TEST_AUTH} TEST_CONTENT_TYPE=${TEST_CONTENT_TYPE} TEST_SSL=${TEST_SSL} TEST_CONNECTION=${TEST_CONNECTION} TEST_CVERSION=${TEST_CVERSION}";

echo "\n"
exit 1