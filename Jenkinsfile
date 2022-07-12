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
        stage('Configure GIT') {
            withCredentials([
                usernamePassword(credentialsId: 'github', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')
            ]) {
                sh 'git config --global --add url."https://${USERNAME}:${PASSWORD}@github.com/".insteadOf "https://github.com/"'
            }
        }

//         stage('Checkout GoDriver') {
//             dir('modules/go-driver') {
//                 sh 'if [ ! -z "${GODRIVER_BRANCH}" ]; then git fetch; git checkout "${GODRIVER_BRANCH}"; fi'
//             }
//         }
        container('worker') {
            stage('Docker Login') {
                withCredentials([
                    file(credentialsId: 'kubernetes-registry-gke-auth', variable: 'AUTH_FILE'),
                    string(credentialsId: 'kubernetes-registry-gke-url', variable: 'AUTH_URL')
                ]) {
                    sh 'cat ${AUTH_FILE} | docker login -u _json_key --password-stdin https://${AUTH_URL}'
                }
            }
            stage('Enable dockerx') {
                sh 'docker buildx create --name builder --driver docker-container --driver-opt network=host --use || echo "Do not recreate"'
            }
            stage('Configure GIT') {
                withCredentials([
                    usernamePassword(credentialsId: 'github', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')
                ]) {
                    sh 'git config --global --add url."https://${USERNAME}:${PASSWORD}@github.com/".insteadOf "https://github.com/"'
                }
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

                    $HOME/resources/itzpapalotl-v1.2.0.zip
                    done

                    if ! [ -f "$HOME/resources/itzpapalotl-v1.2.0.zip" ]; then
                      exit 1
                    fi
                '''
            }
            stage('Run Test') {
                sh 'pwd'
                sh 'ls -l'
                sh 'make run-unit-tests GOIMAGE=gcr.io/gcr-for-testing/golang:1.16.6-stretch VERBOSE=1'
            }
        }
    }
}
