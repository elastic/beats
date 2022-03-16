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
        runBuilds(quietPeriodFactor: 2000, branches: ['main', '8.<minor>', '8.<next-patch>', '7.<minor>'])
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
    // IMPORTANT: ephemeral Orka VMs are not provisioned so, we cannot run builds on daily basis at all.
    // build(quietPeriod: quietPeriod, job: "Beats/beats/${branch}", parameters: [booleanParam(name: 'macosTest', value: true)], wait: false, propagate: false)
    // Increate the quiet period for the next iteration
    quietPeriod += args.quietPeriodFactor
  }
}
