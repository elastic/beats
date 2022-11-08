@Library('apm@current') _

pipeline {
  agent none
  environment {
    NOTIFY_TO = credentials('notify-to')
    PIPELINE_LOG_LEVEL = 'INFO'
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
  }
  triggers {
    cron('H H(1-2) * * 0')
  }
  stages {
    stage('Weekly beats builds for AWS') {
      steps {
        runBuilds(quietPeriodFactor: 1000, branches: ['main', '8.<next-minor>', '8.<minor>', '8.<next-patch>', '7.<minor>'], parameters: [booleanParam(name: 'awsCloudTests', value: true)])
      }
    }
    stage('Weekly beats builds for Orka M1') {
      steps {
        // There are some limitations with the number of concurrent macos m1 that can run in parallel
        // let's only run for the `main` branch for the timebeing and wait to start a bit longer,
        // so the previous stage for AWS validation can run further
        runBuilds(quietPeriodFactor: 10000, branches: ['main'], parameters: [booleanParam(name: 'macosM1Test', value: true)])
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult(prComment: false)
    }
  }
}

def runBuilds(Map args = [:]) {
  def branches = getBranchesFromAliases(aliases: args.branches)

  def quietPeriod = 0
  branches.each { branch ->
    build(quietPeriod: quietPeriod, job: "Beats/beats/${branch}", parameters: args.parameters, wait: false, propagate: false)
    // Increate the quiet period for the next iteration
    quietPeriod += args.quietPeriodFactor
  }
}
