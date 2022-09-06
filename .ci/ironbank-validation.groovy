#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-20 && immutable' }
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    PIPELINE_LOG_LEVEL = "INFO"
    BEATS_FOLDER = "x-pack/heartbeat"
    SLACK_CHANNEL = '#beats'
    NOTIFY_TO = 'ironbank-beats-validation+observability-robots-internal@elastic.co'
  }
  options {
    timeout(time: 31, unit: 'HOURS')
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
    }
  }
  post {
    failure {
      notifyStatus(slackStatus: 'danger', subject: "[${env.REPO}] Ironbank validation failed", body: "(<${env.RUN_DISPLAY_URL}|Open>)")
    }
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
