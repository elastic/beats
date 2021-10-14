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
    cron('H H(2-3) * * 1-5')
  }
  stages {
    stage('Nighly beats builds') {
      steps {
        runBuild(quietPeriod: 0, job: 'Beats/beats/master')
        runBuild(quietPeriod: 2000, job: 'Beats/beats/7.x')
        runBuild(quietPeriod: 4000, job: 'Beats/beats/7.<minor>')
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
  def jobName = args.job
  if (jobName.contains('7.<minor>')) {
    def parts = stackVersions.release().split('\\.')
    jobName = args.job.replaceAll('<minor>', parts[1])
  }
  build(quietPeriod: args.quietPeriod, job: jobName, parameters: [booleanParam(name: 'macosTest', value: true)], wait: false, propagate: false)
}
