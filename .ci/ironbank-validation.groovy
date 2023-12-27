#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-22 && immutable' }
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    PIPELINE_LOG_LEVEL = "INFO"
    BEATS_FOLDER = "x-pack/heartbeat"
    SLACK_CHANNEL = '#ingest-notifications'
    NOTIFY_TO = 'observability-robots-internal+ironbank-beats-validation@elastic.co'
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    disableConcurrentBuilds()
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}")
        setEnvVar("GO_VERSION", readFile("${BASE_DIR}/.go-version").trim())
        dir("${BASE_DIR}"){
          setEnvVar('BEAT_VERSION', sh(label: 'Get beat version', script: 'make get-version', returnStdout: true)?.trim())
        }
      }
    }
    stage('Package'){
      options { skipDefaultCheckout() }
      steps {
        withMageEnv(){
          dir("${env.BASE_DIR}/${env.BEATS_FOLDER}") {
            sh(label: 'make ironbank-package', script: "make -C ironbank package")
          }
        }
      }
      post {
        failure {
          notifyStatus(slackStatus: 'danger', subject: "[${env.REPO}@${BRANCH_NAME}] package for ${env.BEATS_FOLDER}", body: "Contact the heartbeats team. (<${env.RUN_DISPLAY_URL}|Open>)")
        }
      }
    }
    stage('Ironbank'){
      options { skipDefaultCheckout() }
      steps {
        withMageEnv(){
          dir("${env.BASE_DIR}/${env.BEATS_FOLDER}") {
            sh(label: 'mage ironbank', script: 'mage ironbank')
          }
        }
      }
      post {
        failure {
          notifyStatus(slackStatus: 'danger', subject: "[${env.REPO}@${BRANCH_NAME}] Ironbank docker context for ${env.BEATS_FOLDER}", body: "Contact the @observablt-robots-team team. (<${env.RUN_DISPLAY_URL}|Open>)")
        }
      }
    }
    stage('Pre-flight'){
      options { skipDefaultCheckout() }
      environment {
        DOCKER_ELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
        DOCKER_REGISTRY = 'docker.elastic.co'
      }
      steps {
        // verify if the docker registry is available by using the dockerLogin step
        // if so then run the validation which interacts with third party systems
        retryWithSleep(retries: 3, seconds: 10, backoff: true) {
          dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
        }
      }
    }
    stage('Validate'){
      options { skipDefaultCheckout() }
      steps {
        withMageEnv(){
          dir("${env.BASE_DIR}/${env.BEATS_FOLDER}") {
            sh(label: 'make validate-ironbank', script: "make -C ironbank validate-ironbank")
          }
        }
      }
      post {
        failure {
          notifyStatus(slackStatus: 'danger', subject: "[${env.REPO}@${BRANCH_NAME}] Ironbank validation failed", body: "Contact the @observablt-robots-team team. (<${env.RUN_DISPLAY_URL}|Open>)")
        }
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult(prComment: true)
    }
  }
}

def notifyStatus(def args = [:]) {
  releaseNotification(slackChannel: "${env.SLACK_CHANNEL}",
                      slackColor: args.slackStatus,
                      slackCredentialsId: 'jenkins-slack-integration-token',
                      to: "${env.NOTIFY_TO}",
                      subject: args.subject,
                      body: args.body)
}
