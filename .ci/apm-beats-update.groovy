#!/usr/bin/env groovy
@Library('apm@current') _

pipeline {
  agent { label 'linux && immutable' }
  environment {
    REPO = 'apm-server'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    NOTIFY_TO = credentials('notify-to')
    GITHUB_CHECK_ITS_NAME = 'APM Server Beats update'
    PATH = "${env.PATH}:${env.WORKSPACE}/bin"
    HOME = "${env.WORKSPACE}"
    GOPATH = "${env.WORKSPACE}"
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '100', artifactNumToKeepStr: '30', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  triggers {
    issueCommentTrigger('(?i).*/run\\s+(?:apm-beats-update\\W+)?.*')
  }
  stages {
    /**
     Checkout the code and stash it, to use it on other stages.
    */
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "beats")
        script {
          dir("beats"){
            env.GO_VERSION = readFile(".go-version").trim()
            def regexps =[
              "^devtools/mage.*",
              "^libbeat/scripts/Makefile",
            ]
            env.BEATS_UPDATED = isGitRegionMatch(patterns: regexps)

            // Skip all the stages except docs for PR's with asciidoc changes only
            env.ONLY_DOCS = isGitRegionMatch(patterns: [ '.*\\.asciidoc' ], comparator: 'regexp', shouldMatchAll: true)
          }
        }
      }
    }
    /**
    updates beats updates the framework part and go parts of beats.
    Then build and test.
    Finally archive the results.
    */
    stage('Update Beats') {
      options { skipDefaultCheckout() }
      when {
        beforeAgent true
        anyOf {
          branch 'master'
          branch "\\d+\\.\\d+"
          branch "v\\d?"
          tag "v\\d+\\.\\d+\\.\\d+*"
          allOf {
            expression { return env.BEATS_UPDATED != "false" || isCommentTrigger() }
            changeRequest()
          }

        }
      }
      steps {
        withGithubNotify(context: 'Check Apm Server Beats Update') {
          deleteDir()
          gitCheckout(basedir: "${BASE_DIR}",
            repo: "git@github.com:elastic/${REPO}.git",
            branch: 'master',
            credentialsId: 'credentials-id',
            githubNotifyFirstTimeContributor: false,
            depth: 1,
            reference: "/var/lib/jenkins/.git-references/${REPO}.git"
          )
          dir("${BASE_DIR}"){
            sh(label: 'Update Beats script', script: './ci/script/apm-server-update-beats.sh')
          }
        }
      }
      post {
        always {
          catchError(buildResult: 'SUCCESS', message: 'Failed to grab test results tar files', stageResult: 'SUCCESS') {
            tar(file: "update-beats-system-tests-linux-files.tgz", archive: true, dir: "system-tests", pathPrefix: "${BASE_DIR}/build")
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
