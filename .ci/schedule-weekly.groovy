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
    stage('Weekly beats builds') {
      steps {
        runBuild(quietPeriod: 0, branch: 'master')
        runBuild(quietPeriod: 1000, branch: '8.<minor>')
        runBuild(quietPeriod: 2000, branch: '7.<minor>')
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult(prComment: false)
    }
  }
}

def runBuild(Map args = [:]) {
  def branch = args.branch
  // special macro to look for the latest minor version
  if (branch.contains('8.<minor>')) {
    branch = bumpUtils.getMajorMinor(bumpUtils.getCurrentMinorReleaseFor8())
  }
  if (branch.contains('7.<minor>')) {
    branch = bumpUtils.getMajorMinor(bumpUtils.getCurrentMinorReleaseFor7())
  }
  build(quietPeriod: args.quietPeriod, job: "Beats/beats/${branch}", parameters: [booleanParam(name: 'awsCloudTests', value: true)], wait: false, propagate: false)
}
