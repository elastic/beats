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
    cron('H H(1-4) * * 0')
  }
  stages {
    stage('Nighly beats builds') {
      steps {
<<<<<<< HEAD
        build(quietPeriod: 0, job: 'Beats/beats/master', parameters: [booleanParam(name: 'awsCloudTests', value: true), booleanParam(name: 'macosTest', value: true)], wait: false, propagate: false)
        build(quietPeriod: 1000, job: 'Beats/beats/7.x', parameters: [booleanParam(name: 'awsCloudTests', value: true), booleanParam(name: 'macosTest', value: true)], wait: false, propagate: false)
=======
        runBuilds(quietPeriodFactor: 1000, branches: ['main', '8.<minor>', '7.<minor>', '7.<next-minor>'])
>>>>>>> 237937085a (use main default branch (#29710))
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult(prComment: false)
    }
  }
}
