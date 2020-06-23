#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'linux && immutable' }
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    DOCKER_REGISTRY = 'docker.elastic.co'
    DOCKER_REGISTRY_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    GOPATH = "${env.WORKSPACE}"
    HOME = "${env.WORKSPACE}"
    JOB_GCS_BUCKET = credentials('gcs-bucket')
    NOTIFY_TO = credentials('notify-to')
    PATH = "${env.GOPATH}/bin:${env.PATH}"
    PIPELINE_LOG_LEVEL='INFO'
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  triggers {
    cron 'H H(0-5) * * 1-5'
  }
  parameters {
    booleanParam(name: "RELEASE_TEST_IMAGES", defaultValue: "true", description: "If it's needed to build & push Beats' test images")
    string(name: 'BRANCH_REFERENCE', defaultValue: "master", description: "Git branch/tag to use")
  }
  stages {
    stage('Checkout') {
      steps {
        gitCheckout(basedir: "${BASE_DIR}",
          branch: "${params.BRANCH_REFERENCE}",
          repo: "https://github.com/elastic/${REPO}.git",
          credentialsId: "${JOB_GIT_CREDENTIALS}"
        )
        dir("${BASE_DIR}"){
          setEnvVar("GO_VERSION", readFile(".go-version").trim())
        }
      }
    }
    stage('Install dependencies') {
      when {
        expression { return params.RELEASE_TEST_IMAGES }
      }
      steps {
        sh(label: 'Install virtualenv', script: 'pip install --user virtualenv')
      }
    }
    stage('Metricbeat Test Docker images'){
      options {
        warnError('Metricbeat Test Docker images failed')
      }
      when {
        expression { return params.RELEASE_TEST_IMAGES }
      }
      steps {
        dockerLogin(secret: "${env.DOCKER_REGISTRY_SECRET}", registry: "${env.DOCKER_REGISTRY}")
        dir("${HOME}/${BASE_DIR}"){
          retry(3){
            sh(label: 'Build ', script: ".ci/scripts/build-beats-integrations-test-images.sh '${GO_VERSION}' '${HOME}/${BASE_DIR}/metricbeat'")
          }
        }
      }
    }
    stage('Metricbeat x-pack Test Docker images'){
      options {
        warnError('Metricbeat x-pack Docker images failed')
      }
      when {
        expression { return params.RELEASE_TEST_IMAGES }
      }
      steps {
        dockerLogin(secret: "${env.DOCKER_REGISTRY_SECRET}", registry: "${env.DOCKER_REGISTRY}")
        dir("${HOME}/${BASE_DIR}"){
          retry(3){
            sh(label: 'Build ', script: ".ci/scripts/build-beats-integrations-test-images.sh '${GO_VERSION}' '${HOME}/${BASE_DIR}/x-pack/metricbeat'")
          }
        }
      }
    }
    stage('Filebeat x-pack Test Docker images'){
      options {
        warnError('Filebeat x-pack Test Docker images failed')
      }
      when {
        expression { return params.RELEASE_TEST_IMAGES }
      }
      steps {
        dockerLogin(secret: "${env.DOCKER_REGISTRY_SECRET}", registry: "${env.DOCKER_REGISTRY}")
        dir("${HOME}/${BASE_DIR}"){
          retry(3){
            sh(label: 'Build ', script: ".ci/scripts/build-beats-integrations-test-images.sh '${GO_VERSION}' '${HOME}/${BASE_DIR}/x-pack/filebeat'")
          }
        }
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult()
    }
  }
}
