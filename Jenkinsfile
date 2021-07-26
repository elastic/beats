#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-18 && immutable' }
  environment {
    PIPELINE_LOG_LEVEL = 'INFO'
  }
  options {
    timeout(time: 4, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '60', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
  }
  stages {
    stage('windows-10-20') {
      options { skipDefaultCheckout() }
      steps {
        runBuildAndTest(number: 20, labels: 'windows-10')
      }
    }
    stage('windows-10-40') {
      options { skipDefaultCheckout() }
      steps {
        runBuildAndTest(number: 40, labels: 'windows-10')
      }
    }
    stage('windows-10-80') {
      options { skipDefaultCheckout() }
      steps {
        runBuildAndTest(number: 80, labels: 'windows-10')
      }
    }
    stage('windows-10-120') {
      options { skipDefaultCheckout() }
      steps {
        runBuildAndTest(number: 120, labels: 'windows-10')
      }
    }
  }
}

def runBuildAndTest(Map args = [:]) {
  def mapParallelTasks = [:]
  for(int k = 0;k<args.number;k++) {
    mapParallelTasks["${k}"] = {
                                  withNode(labels: args.labels, forceWorkspace: true) {
                                    sleep randomNumber(min: 2, max: 10)
                                    echo 'done'
                                  }
                                }
  }
  parallel(mapParallelTasks)
}
