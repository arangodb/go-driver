podTemplate(
  containers: [
    containerTemplate(name: 'worker', image: 'eu.gcr.io/hale-ivy-241313/jenkins-worker:2022-02-01.10-15', command: 'sleep', args: '99d')
  ],
  volumes: [
    persistentVolumeClaim(claimName: 'jenkins-go-ebs', mountPath: '/.go'),
    hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock')
  ],
  serviceAccount: 'jenkins-agent',
  ) {
    node(POD_LABEL) {
        stage('Clone') {
            checkout scm
        }

        container('worker') {

            stage('Prepare ENV') {
                sh '''
                    mkdir -p /.go/resources
                    for i in {0..3}
                    do

                    if ! [ -f "/.go/resources/itzpapalotl-v1.2.0.zip" ]; then
                      curl -L0 -o /.go/resources/itzpapalotl-v1.2.0.zip "https://github.com/arangodb-foxx/demo-itzpapalotl/archive/v1.2.0.zip"
                    fi

                    SHA=$(sha256sum /.go/resources/itzpapalotl-v1.2.0.zip | cut -f 1 -d " ")
                    if [ "${SHA}" = "86117db897efe86cbbd20236abba127a08c2bdabbcd63683567ee5e84115d83a" ]; then
                      break
                    fi
                    done

                    if ! [ -f "/.go/resources/itzpapalotl-v1.2.0.zip" ]; then
                      exit 1
                    fi
                '''
            }

            stage('Run Test') {
                sh 'make run-unit-tests GOIMAGE=gcr.io/gcr-for-testing/golang:1.16.6-stretch TEST_RESOURCES="/.go/resources/" VERBOSE=1'
            }
        }
    }
}
