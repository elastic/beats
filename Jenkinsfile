#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-18 && immutable' }
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    PIPELINE_LOG_LEVEL = 'INFO'
    GOPATH = "${env.WORKSPACE}"
    HOME = "${env.WORKSPACE}"
    PATH = "${PATH}:${HOME}/bin"
    JOB_GCS_BUCKET_STASH = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    timestamps()
  }
  triggers {
    cron('H */3 * * *')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true)
        stashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
        dir("${BASE_DIR}"){
          setEnvVar('GO_VERSION', readFile(".go-version").trim())
        }
      }
    }
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
          stage('prepare') {
            options { skipDefaultCheckout() }
            steps {
              deleteDir()
              unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
            }
          }
          stage('build') {
            options { skipDefaultCheckout() }
            steps {
              runCommand('mage build', "${beat}")
            }
          }
          stage('test') {
            options { skipDefaultCheckout() }
            steps {
              runCommand('mage run', "${beat}")
            }
          }
        }
      }
    }
  }
  post {
    cleanup {
      deleteDir()
    }
  }
}

def runCommand(command, beat) {
  withMageEnv(version: env.GO_VERSION) {
    dir("${BASE_DIR}/${beat}"){
      cmd(label: "${command}", script: "${command} || true")
    }
  }
}
