#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'orka && darwin && poc' }
  environment {
    AWS_ACCOUNT_SECRET = 'secret/observability-team/ci/elastic-observability-aws-account-auth'
    AWS_REGION = "${params.awsRegion}"
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    DOCKERHUB_SECRET = 'secret/observability-team/ci/elastic-observability-dockerhub'
    DOCKER_ELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_COMPOSE_VERSION = "1.21.0"
    DOCKER_REGISTRY = 'docker.elastic.co'
    JOB_GCS_BUCKET = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    JOB_GCS_EXT_CREDENTIALS = 'beats-ci-gcs-plugin-file-credentials'
    OSS_MODULE_PATTERN = '^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
    PIPELINE_LOG_LEVEL = 'INFO'
    PYTEST_ADDOPTS = "${params.PYTEST_ADDOPTS}"
    RUNBLD_DISABLE_NOTIFICATIONS = 'true'
    SLACK_CHANNEL = "#beats-build"
    SNAPSHOT = 'true'
    TERRAFORM_VERSION = "0.13.7"
    XPACK_MODULE_PATTERN = '^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
  }
  options {
    timeout(time: 6, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '60', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
        deleteDir()
        // Here we do a checkout into a temporary directory in order to have the
        // side-effect of setting up the git environment correctly.
        gitCheckout(basedir: "${pwd(tmp: true)}", githubNotifyFirstTimeContributor: true)
      }
    }
  }
  post {
    cleanup {
      dir("${BASE_DIR}"){
        notifyBuildResult(prComment: true)
      }
    }
  }
}
