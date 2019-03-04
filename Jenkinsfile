#!/usr/bin/env groovy

library identifier: 'apm@v1.0.2',
retriever: modernSCM(
  [$class: 'GitSCMSource',
  credentialsId: 'f94e9298-83ae-417e-ba91-85c279771570',
  id: '37cf2c00-2cc7-482e-8c62-7bbffef475e2',
  remote: 'git@github.com:elastic/apm-pipeline-library.git'])

pipeline {
  agent none
  environment {
    BASE_DIR="src/github.com/elastic/beats"
  }
  options {
    timeout(time: 1, unit: 'HOURS') 
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
  }
  parameters {
    booleanParam(name: 'Run_As_Master_Branch', defaultValue: false, description: 'Allow to run any steps on a PR, some steps normally only run on master branch.')
  }
  stages {
    /**
     Checkout the code and stash it, to use it on other stages.
    */
    stage('Checkout') {
      agent { label 'linux && immutable' }
      environment {
        PATH = "${env.PATH}:${env.WORKSPACE}/bin"
        HOME = "${env.WORKSPACE}"
        GOPATH = "${env.WORKSPACE}"
      }
      options { skipDefaultCheckout() }
      steps {
        dir("${BASE_DIR}"){
            checkout scm
        }
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
        script {
          env.GO_VERSION = readFile("${BASE_DIR}/.go-version")
        }
      }
    }
    /**
    Updating generated files for Beat.
    Checks the GO environment.
    Checks the Python environment.
    Checks YAML files are generated. 
    Validate that all updates were committed.
    */
    stage('Intake') { 
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
      environment {
        PATH = "${env.PATH}:${env.WORKSPACE}/bin"
        HOME = "${env.WORKSPACE}"
        GOPATH = "${env.WORKSPACE}"
      }
      steps {
        withEnvWrapper() {
          unstash 'source'
          dir("${BASE_DIR}"){
            sh './dev-tools/jenkins_intake.sh'
          }
        }
      }
    }
    stage('Test') {
      failFast true
      parallel {
        /**
          Run unit tests and report junit results.
        */
        stage('Filebeat') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
          environment {
            PATH = "${env.PATH}:${env.WORKSPACE}/bin"
            HOME = "${env.WORKSPACE}"
            GOPATH = "${env.WORKSPACE}"
          }
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){
                sh './filebeat/scripts/jenkins/unit-test.sh'
              }
            }
          }
          post {
            always {
              junit(allowEmptyResults: true,
                keepLongStdio: true,
                testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
      }
    }
  }
  post {
    success {
      echoColor(text: '[SUCCESS]', colorfg: 'green', colorbg: 'default')
    }
    aborted {
      echoColor(text: '[ABORTED]', colorfg: 'magenta', colorbg: 'default')
    }
    failure { 
      echoColor(text: '[FAILURE]', colorfg: 'red', colorbg: 'default')
      //step([$class: 'Mailer', notifyEveryUnstableBuild: true, recipients: "${NOTIFY_TO}", sendToIndividuals: false])
    }
    unstable {
      echoColor(text: '[UNSTABLE]', colorfg: 'yellow', colorbg: 'default')
    }
  }
}
