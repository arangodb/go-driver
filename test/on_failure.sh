#!/bin/bash
echo "failure!\n"

echo "\nARANGODB-STARTER logs: "
docker logs ${TESTCONTAINER}-s

echo -n "\nARANGODB-S-* logs:"
docker ps -f name=${TESTCONTAINER}-s- --format "{{.ID}} {{.Names}}" | xargs -L1 bash -c 'echo -e "\n\tLogs from $1:"; docker logs $0'

if [ $DUMP_ON_FAILURE -eq 1 ] && [ "${TEST_MODE}" != "single" ]; then
    echo "\nAgency dump..."
    AGENCY_CONTATIONER_NAME='agent'
    ANY_CONNECTION_HTTPS=$(docker ps -f name=${TESTCONTAINER}-s-${AGENCY_CONTATIONER_NAME}  --format '{{.Names}}'  | head -n 1 | sed -E 's/.*-([a-zA-Z0-9.-]+)-([0-9]+)$/https:\/\/\1:\2/')
    LEADER_CONNECTION_SSL=$(curl -Lk --no-progress-meter $ANY_CONNECTION_HTTPS/_api/agency/config | jq -r '.configuration.pool[.leaderId]')
    LEADER_CONNECTION_HTTPS=$(echo $LEADER_CONNECTION_SSL | sed 's/^ssl:\/\//https:\/\//')
    DUMP_FOLDER_PATH=./arango_data_v$(cat ./v2/version/VERSION)
    mkdir -p $DUMP_FOLDER_PATH
    DUMP_FILE_PATH=$DUMP_FOLDER_PATH/FAIL_agency_dump-HTTP_VPACK.json
    echo $(curl -Lk --no-progress-meter $LEADER_CONNECTION_HTTPS/_api/agency/state) > $DUMP_FILE_PATH
    echo "Agency dump created at $(realpath $DUMP_FILE_PATH)"
fi

echo "\nV${MAJOR_VERSION} Tests with ARGS: TEST_MODE=${TEST_MODE} TEST_AUTH=${TEST_AUTH} TEST_CONTENT_TYPE=${TEST_CONTENT_TYPE} TEST_SSL=${TEST_SSL} TEST_CONNECTION=${TEST_CONNECTION} TEST_CVERSION=${TEST_CVERSION}";

echo "\n"
exit 1