#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    PIPELINE_LOG_LEVEL = "INFO"
    BEATS_TESTER_JOB = 'Beats/beats-tester-mbp/main'
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
  triggers {
    issueCommentTrigger('(?i)^\\/beats-tester$')
    upstream("Beats/packaging/${env.JOB_BASE_NAME}")
  }
  stages {
    stage('Filter build') {
      agent { label 'ubuntu-20' }
      when {
        beforeAgent true
        anyOf {
          triggeredBy cause: "IssueCommentCause"
          expression {
            def ret = isUserTrigger() || isUpstreamTrigger()
            if(!ret){
              currentBuild.result = 'NOT_BUILT'
              currentBuild.description = "The build has been skipped"
              currentBuild.displayName = "#${BUILD_NUMBER}-(Skipped)"
              echo("the build has been skipped due the trigger is a branch scan and the allow ones are manual, GitHub comment, and upstream job")
            }
            return ret
          }
        }
      }
      stages {
        stage('Checkout') {
          options { skipDefaultCheckout() }
          steps {
            deleteDir()
            gitCheckout(basedir: "${BASE_DIR}")
            setEnvVar('VERSION', sh(script: "grep ':stack-version:' ${BASE_DIR}/libbeat/docs/version.asciidoc | cut -d' ' -f2", returnStdout: true).trim())
          }
        }
        stage('Build main') {
          options { skipDefaultCheckout() }
          when { branch 'main' }
          steps {
            runBeatsTesterJob(version: "${env.VERSION}-SNAPSHOT")
          }
        }
        stage('Build *.x branch') {
          options { skipDefaultCheckout() }
          when { branch '*.x' }
          steps {
            runBeatsTesterJob(version: "${env.VERSION}-SNAPSHOT")
          }
        }
        stage('Build PullRequest') {
          options { skipDefaultCheckout() }
          when { changeRequest() }
          steps {
            runBeatsTesterJob(version: "${env.VERSION}-SNAPSHOT",
                              apm: "https://storage.googleapis.com/apm-ci-artifacts/jobs/pull-requests/pr-${env.CHANGE_ID}",
                              beats: "https://storage.googleapis.com/beats-ci-artifacts/pull-requests/pr-${env.CHANGE_ID}")
          }
        }
        stage('Build release branch') {
          options { skipDefaultCheckout() }
          when {
            not {
              anyOf {
                branch comparator: 'REGEXP', pattern: '(main|.*x)'
                changeRequest()
              }
            }
           }
          steps {
            // TODO: to use the git commit that triggered the upstream build
            runBeatsTesterJob(version: "${env.VERSION}-SNAPSHOT")
          }
        }
      }
    }
  }
}

def runBeatsTesterJob(Map args = [:]) {
  def apm = args.get('apm', '')
  def beats = args.get('beats', '')
  def version = args.version

  if (isUpstreamTrigger()) {
    copyArtifacts(filter: 'beats-tester.properties',
                  flatten: true,
                  projectName: "Beats/packaging/${env.JOB_BASE_NAME}",
                  selector: upstream(fallbackToLastSuccessful: true))
    def props = readProperties(file: 'beats-tester.properties')
    apm = props.get('APM_URL_BASE', '')
    beats = props.get('BEATS_URL_BASE', '')
    version = props.get('VERSION', '8.0.0-SNAPSHOT')
  }
  if (apm?.trim() || beats?.trim()) {
    build(job: env.BEATS_TESTER_JOB, propagate: false, wait: false,
          parameters: [
            string(name: 'APM_URL_BASE', value: apm),
            string(name: 'BEATS_URL_BASE', value: beats),
            string(name: 'VERSION', value: version)
          ])
  } else {
    build(job: env.BEATS_TESTER_JOB, propagate: false, wait: false, parameters: [ string(name: 'VERSION', value: version) ])
  }
}
