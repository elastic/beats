#!/usr/bin/env groovy
@Library('apm@current') _

pipeline {
  agent { label 'linux && immutable' }
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
          // golang(){
          //   dir("${BEATS_DIR}"){
          //     sh(label: 'Get Beats module', script: "go get -v ${BEATS_MOD}")
          //   }
          // }
          dir("${BASE_DIR}"){
            git(credentialsId: 'f6c7695a-671e-4f4f-a331-acdce44ff9ba',
              url:  "git@github.com:elastic/${REPO}.git")
          }
          golang(){
            dir("${BASE_DIR}"){
              sh(label: 'Update Beats script', script: """
                export BEATS_VERSION=${env.GIT_BASE_COMMIT}
                git config --global user.email "you@example.com"
                git config --global user.name "Your Name"
                go build -o build/mage github.com/magefile/mage
                export PATH=\${WORKSPACE}/\${BASE_DIR}/build:\${PATH}
                mage pythonEnv
                . build/ve/linux/bin/activate
                go mod edit -replace github.com/elastic/beats/v7=${env.GOPATH}/src/github.com/elastic/beats-local
                git commit -m beats-update go.mod # commit locally to prevent "Error: some files are not up-to-date."
                make update
                git commit -a -m "update"
                make check
                git config --global --add remote.origin.fetch "+refs/pull/*/head:refs/remotes/origin/pr/*"
                make update-beats
              """)
            }
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
  // post {
  //   cleanup {
  //     notifyBuildResult()
  //   }
  // }
}

def golang(Closure body){
  docker.image("golang:${GO_VERSION}").inside(){
    body()
  }
}
