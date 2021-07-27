#!/usr/bin/env groovy
@Library('apm@current') _

pipeline {
  agent none
  options {
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
  }
  stages {
    stage('windows-10-20') {
      steps { runBuildAndTest(number: 20, labels: 'windows-10') }
    }
    stage('windows-10-40') {
      steps { runBuildAndTest(number: 40, labels: 'windows-10') }
    }
    stage('windows-10-80') {
      steps { runBuildAndTest(number: 80, labels: 'windows-10') }
    }
    stage('windows-10-120') {
      steps { runBuildAndTest(number: 120, labels: 'windows-10') }
    }
  }
}

def runBuildAndTest(Map args = [:]) {
  def mapParallelTasks = [:]
  for(int k = 0;k<args.number;k++) {
    mapParallelTasks["${k}"] = { withNode(labels: args.labels, forceWorkspace: true) {
      sleep randomNumber(min: 2, max: 10)
      bat 'ECHO Hello' 
      bat 'type nul > your_file.txt' }
    }
  }
  parallel(mapParallelTasks)
}
