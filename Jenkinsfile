podTemplate(
  containers: [
    containerTemplate(name: 'worker', image: 'gcr.io/gcr-for-testing/kube-arangodb/cicd:2022-06-27.22-55', command: 'sleep', args: '99d')
  ],
  volumes: [
    hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock')
  ],
  serviceAccount: 'jenkins-agent',
  ) {
    node(POD_LABEL) {
        stage('Clone') {
            checkout scm
        }

        container('worker') {
            stage('Find myself') {
                sh 'docker ps'
                sh 'env'
            }

            stage('Prepare ENV') {
                sh '''
                    mkdir -p $HOME/resources
                    for i in {0..3}
                    do

                    if ! [ -f "$HOME/resources/itzpapalotl-v1.2.0.zip" ]; then
                      curl -L0 -o $HOME/resources/itzpapalotl-v1.2.0.zip "https://github.com/arangodb-foxx/demo-itzpapalotl/archive/v1.2.0.zip"
                    fi

                    SHA=$(sha256sum $HOME/resources/itzpapalotl-v1.2.0.zip | cut -f 1 -d " ")
                    if [ "${SHA}" = "86117db897efe86cbbd20236abba127a08c2bdabbcd63683567ee5e84115d83a" ]; then
                      break
                    fi
                    done

                    if ! [ -f "$HOME/resources/itzpapalotl-v1.2.0.zip" ]; then
                      exit 1
                    fi
                '''
            }

            stage('Run Test') {
                sh 'make run-tests-single GOIMAGE=gcr.io/gcr-for-testing/golang:1.16.6-stretch STARTER=gcr.io/gcr-for-testing/arangodb/arangodb-starter:latest ALPINE_IMAGE=gcr.io/gcr-for-testing/alpine:3.4 ARANGODB=eu.gcr.io/arangodb-ci/official/arangodb/arangodb:3.6.16 VERBOSE=1'
            }
        }
    }
}
