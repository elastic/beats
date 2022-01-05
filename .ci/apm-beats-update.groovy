#!/usr/bin/env groovy
@Library('apm@current') _

pipeline {
  agent { label 'master' }
  environment {
    REPO = 'apm-server'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    BEATS_MOD = 'github.com/elastic/beats-local'
    BEATS_DIR = "src/${BEATS_MOD}"
    NOTIFY_TO = credentials('notify-to')
    GITHUB_CHECK_ITS_NAME = 'APM Server Beats update'
    PATH = "${env.PATH}:${env.WORKSPACE}/bin"
    HOME = "${env.WORKSPACE}"
    GOPATH = "${env.WORKSPACE}"
    SHELL = "/bin/bash"
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
    upstream("Beats/beats/${ env.JOB_BASE_NAME.startsWith('PR-') ? 'none' : env.JOB_BASE_NAME }")
  }
  stages {
    stage('Filter build') {
      agent { label 'ubuntu-18 && immutable' }
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
        /**
        Checkout the code and stash it, to use it on other stages.
        */
        stage('Checkout') {
          options { skipDefaultCheckout() }
          steps {
            deleteDir()
            gitCheckout(basedir: "${BEATS_DIR}", githubNotifyFirstTimeContributor: false)
            script {
              dir("${BEATS_DIR}"){
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
              branch 'main'
              branch "\\d+\\.\\d+"
              branch "v\\d?"
              tag "v\\d+\\.\\d+\\.\\d+*"
              allOf {
                expression { return env.BEATS_UPDATED != "false" || isCommentTrigger() || isUserTrigger() }
                changeRequest()
              }
            }
          }
          steps {
            withGithubNotify(context: 'Check Apm Server Beats Update') {
              beatsUpdate()
            }
          }
        }
      }
    }
  }
}

def beatsUpdate() {
  def os = "linux"
  def goRoot = "${env.WORKSPACE}/.gvm/versions/go${GO_VERSION}.${os}.amd64"

  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "GOROOT=${goRoot}",
    "PATH=${env.WORKSPACE}/bin:${goRoot}/bin:${env.PATH}",
    "MAGEFILE_CACHE=${env.WORKSPACE}/.magefile",
  ]) {
    dir("${BEATS_DIR}") {
      sh(label: "Create branch localVersion", script: "git checkout -b localVersion")
      sh(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.sh")
    }
    dir("${BASE_DIR}"){
      git(credentialsId: 'f6c7695a-671e-4f4f-a331-acdce44ff9ba',
        url:  "git@github.com:elastic/${env.REPO}.git")
      sh(label: 'Update Beats script', script: """
        git config --global user.email "none@example.com"
        git config --global user.name "None"
        git config --global --add remote.origin.fetch "+refs/pull/*/head:refs/remotes/origin/pr/*"

        go mod edit -replace github.com/elastic/beats/v7=\${GOPATH}/src/github.com/elastic/beats-local
        go mod tidy
        echo '{"name": "${GOPATH}/src/github.com/elastic/beats-local", "licenceType": "Elastic"}' >> \${GOPATH}/src/github.com/elastic/beats-local/dev-tools/notice/overrides.json

        make update
        git commit -a -m beats-update

        make check
      """)
    }
  }
}
