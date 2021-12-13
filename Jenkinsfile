#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'orka && darwin && poc' }
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    PIPELINE_LOG_LEVEL = 'INFO'
    COMMAND = "${params.COMMAND}"
    GOPATH = "${env.WORKSPACE}"
    HOME = "${env.WORKSPACE}"
    PATH = "${PATH}:${HOME}/bin"
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    timestamps()
  }
  parameters {
    string(name: 'COMMAND', defaultValue: 'mage build', description: 'What command?')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true)
        dir("${BASE_DIR}"){
          setEnvVar('GO_VERSION', readFile(".go-version").trim())
          sh '.ci/scripts/install-go.sh'
        }
      }
    }
    stage('Run'){
      options { skipDefaultCheckout() }
      steps {
        withMageEnv(version: env.GO_VERSION) {
          runBeats()
        }
      }
    }
  }
}

def runBeats() {
  def beats= ["auditbeat",
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
              "x-pack/packetbeat"]
  beats.each { beat ->
    stage(beat) {
      dir("${BASE_DIR}/${beat}"){
        cmd(label: "${env.COMMAND}", script: "${env.COMMAND} || true")
      }
    }
  }
}
