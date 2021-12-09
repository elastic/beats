#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    PIPELINE_LOG_LEVEL = 'INFO'
    GOPATH = "${env.WORKSPACE}"
    HOME = "${env.WORKSPACE}"
    PATH = "${PATH}:${HOME}/bin"
  }
  options {
    timestamps()
  }
  stages {
    stage('Run'){
      options { skipDefaultCheckout() }
      matrix {
        agent { label 'orka && darwin && poc' }
        axes {
            axis {
                name 'beat'
                values "auditbeat",
                       "filebeat",
                       "heartbeat",
                       "libbeat",
                       "metricbeat",
                       "packetbeat",
                       "x-pack/auditbeat",
                       "x-pack/elastic-agent",
                       "x-pack/filebeat",
                       "x-pack/functionbeat",
                       "x-pack/heartbeat",
                       "x-pack/libbeat",
                       "x-pack/metricbeat",
                       "x-pack/osquerybeat",
                       "x-pack/packetbeat"
            }
        }
        stages {
          stage('Checkout') {
            options { skipDefaultCheckout() }
            steps {
              deleteDir()
              gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true)
              dir("${BASE_DIR}"){
                setEnvVar('GO_VERSION', readFile(".go-version").trim())
              }
            }
          }
          stage('build') {
            steps {
              runCommand('mage build', "${beat}")
            }
          }
          stage('test') {
            steps {
              runCommand('mage run', "${beat}")
            }
          }
        }
      }
    }
  }
}

def runCommand(command, beat) {
  dir("${BASE_DIR}/${beat}"){
    cmd(label: "${command}", script: "${command} || true")
  }
}

/**
* Tear down the setup for the permanent workers.
*/
def tearDown() {
  catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
    cmd(label: 'Remove the entire module cache', script: 'go clean -modcache', returnStatus: true)
    fixPermissions("${WORKSPACE}")
    // IMPORTANT: Somehow windows workers got a different opinion regarding removing the workspace.
    //            Windows workers are ephemerals, so this should not really affect us.
    if (isUnix()) {
      dir("${WORKSPACE}") {
        deleteDir()
      }
    }
  }
}

/**
* This method fixes the filesystem permissions after the build has happenend. The reason is to
* ensure any non-ephemeral workers don't have any leftovers that could cause some environmental
* issues.
*/
def fixPermissions(location) {
  if(isUnix()) {
    try {
      timeout(5) {
        sh(label: 'Fix permissions', script: """#!/usr/bin/env bash
          set +x
          echo "Cleaning up ${location}"
          source ./dev-tools/common.bash
          docker_setup
          script/fix_permissions.sh ${location}""", returnStatus: true)
      }
    } catch (Throwable e) {
      echo "There were some failures when fixing the permissions. ${e.toString()}"
    }
  }
}
