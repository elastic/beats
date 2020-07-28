#!/usr/bin/env groovy

@Library('apm@feature/2.0-beats') _

pipeline {
  agent { label 'ubuntu-18 && immutable' }
  environment {
    AWS_ACCOUNT_SECRET = 'secret/observability-team/ci/elastic-observability-aws-account-auth'
    BASE_DIR = 'src/github.com/elastic/beats'
    DOCKERELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_COMPOSE_VERSION = "1.21.0"
    DOCKER_REGISTRY = 'docker.elastic.co'
    GOX_FLAGS = "-arch amd64"
    JOB_GCS_BUCKET = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    PIPELINE_LOG_LEVEL = 'INFO'
    OSS_MODULE_PATTERN = '^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
    RUNBLD_DISABLE_NOTIFICATIONS = 'true'
    TERRAFORM_VERSION = "0.12.24"
    XPACK_MODULE_PATTERN = '^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
  }
  triggers {
    issueCommentTrigger('(?i)(.*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*|^/test\\W+.*$)')
  }
  parameters {
    booleanParam(name: 'allCloudTests', defaultValue: false, description: 'Run all cloud integration tests.')
    booleanParam(name: 'awsCloudTests', defaultValue: false, description: 'Run AWS cloud integration tests.')
    string(name: 'awsRegion', defaultValue: 'eu-central-1', description: 'Default AWS region to use for testing.')
    booleanParam(name: 'runAllStages', defaultValue: false, description: 'Allow to run all stages.')
    booleanParam(name: 'macosTest', defaultValue: false, description: 'Allow macOS stages.')
    booleanParam(name: 'windowsTest', defaultValue: true, description: 'Allow Windows stages.')
    booleanParam(name: 'debug', defaultValue: false, description: 'Allow debug logging for Jenkins steps')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true)
        stashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
        script {
          dir("${BASE_DIR}"){
            // Skip all the stages except docs for PR's with asciidoc and md changes only
            env.ONLY_DOCS = isGitRegionMatch(patterns: [ '.*\\.(asciidoc|md)' ], shouldMatchAll: true)
          }
        }
      }
    }
    stage('Lint'){
      options { skipDefaultCheckout() }
      environment {
        GOFLAGS = '-mod=readonly'
      }
      when {
        anyOf {
          not { changeRequest() }                           // If no PR
          allOf {                                           // If PR and no docs changes
            expression { return env.ONLY_DOCS == "false" }
            changeRequest()
          }
          expression { return params.runAllStages }         // If UI forced
        }
      }
      steps {
        echo 'linting'
      }
    }
    stage('Build&Test') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        unstash "source"
        dir("${BASE_DIR}"){
          script {
            def masterFile = 'Jenkinsfile.yml'
            def changeset = readYaml(file: masterFile)['changeset']
            readYaml(file: masterFile)['projects'].each { projectName ->
              generateStages(project: projectName, changeset: changeset).each { k,v ->
                mapParallelTasks["${k}"] = v
              }
            }
            parallel(mapParallelTasks)
          }
        }
      }
      post {
        always {
          echo 'TODO: save markdown files for the record'
        }
      }
    }
  }
  post {
    // TODO: Support runbld
    cleanup {
      notifyBuildResult(prComment: true)
    }
  }
}
