#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    PIPELINE_LOG_LEVEL = "INFO"
    BEATS_TESTER_JOB = 'Beats/beats-tester-mbp/main'
    BASE_URL = "https://staging.elastic.co/${params.version}/downloads"
    APM_BASE_URL = "${env.BASE_URL}/apm-server"
    BEATS_BASE_URL = "${env.BASE_URL}/beats"
    VERSION = "${params.version?.split('-')[0]}"
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    disableConcurrentBuilds()
  }
  parameters {
    string(name: 'version', defaultValue: '', description: 'Id of the Build Candidate (7.10.0-b55684ff).')
    string(name: 'BRANCH_REFERENCE', defaultValue: 'master', description: 'Branch to grab the Groovy script(for test changes).')
  }
  stages {
    stage('Run Beat Tester') {
      options { skipDefaultCheckout() }
      when {
        expression {
          return '' != "${VERSION}"
        }
      }
      steps {
        build(job: env.BEATS_TESTER_JOB, propagate: true, wait: true,
          parameters: [
            string(name: 'APM_URL_BASE', value: "${APM_BASE_URL}"),
            string(name: 'BEATS_URL_BASE', value: "${BEATS_BASE_URL}"),
            string(name: 'VERSION', value: "${VERSION}")
        ])
      }
    }
  }
}
