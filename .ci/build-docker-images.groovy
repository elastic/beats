#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-20' }
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    DOCKER_REGISTRY = 'docker.elastic.co'
    DOCKER_REGISTRY_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    HOME = "${env.WORKSPACE}"
    NOTIFY_TO = credentials('notify-to')
    PIPELINE_LOG_LEVEL = 'INFO'
    JOB_GIT_CREDENTIALS = "f6c7695a-671e-4f4f-a331-acdce44ff9ba"
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
    string(name: 'BRANCH_REFERENCE', defaultValue: "main", description: "Git branch/tag to use")
  }
  stages {
    stage('Checkout') {
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}",
          branch: "${params.BRANCH_REFERENCE}",
          repo: "https://github.com/elastic/${REPO}.git",
          credentialsId: "${JOB_GIT_CREDENTIALS}"
        )
        dir("${BASE_DIR}"){
          setEnvVar("GO_VERSION", readFile(file: ".go-version")?.trim())
        }
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
        withMageEnv(){
          dir("${BASE_DIR}/metricbeat"){
            retryWithSleep(retries: 3, seconds: 5, backoff: true){
              sh(label: 'Build', script: "mage compose:buildSupportedVersions");
              sh(label: 'Push', script: "mage compose:pushSupportedVersions");
            }
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
        withMageEnv(){
          dir("${BASE_DIR}/x-pack/metricbeat"){
            retryWithSleep(retries: 3, seconds: 5, backoff: true){
              sh(label: 'Build', script: "mage compose:buildSupportedVersions");
              sh(label: 'Push', script: "mage compose:pushSupportedVersions");
            }
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
        withMageEnv(){
          dir("${BASE_DIR}/x-pack/filebeat"){
            retryWithSleep(retries: 3, seconds: 5, backoff: true){
              sh(label: 'Build', script: "mage compose:buildSupportedVersions");
              sh(label: 'Push', script: "mage compose:pushSupportedVersions");
            }
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
