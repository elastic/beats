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
        runBuild(quietPeriod: 0, job: 'Beats/beats/main')
        // This should be `current_8` bump.getCurrentMinorReleaseFor8
        runBuild(quietPeriod: 2000, job: 'Beats/beats/8.0')
        // This should be `current_7` bump.getCurrentMinorReleaseFor7 or
        // `next_minor_7`  bump.getNextMinorReleaseFor7
        runBuild(quietPeriod: 4000, job: 'Beats/beats/7.17')
        runBuild(quietPeriod: 6000, job: 'Beats/beats/7.16')
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
  build(quietPeriod: args.quietPeriod, job: jobName, parameters: [booleanParam(name: 'macosTest', value: true)], wait: false, propagate: false)
}
