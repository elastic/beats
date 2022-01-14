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
        runBuilds(quietPeriodFactor: 1000, branches: ['master', '8.<minor>', '7.<minor>', '7.<next-minor>'])
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
  def branches = []
  // Expand macros and filter duplicated matches.
  args.branches.each { branch ->
    def branchName = getBranchName(branch)
    if (!branches.contains(branchName)) {
      branches << branchName
    }
  }

  def quietPeriod = 0
  branches.each { branch ->
    build(quietPeriod: quietPeriod, job: "Beats/beats/${branch}", parameters: [booleanParam(name: 'awsCloudTests', value: true)], wait: false, propagate: false)
    // Increate the quiet period for the next iteration
    quietPeriod += args.quietPeriodFactor
  }
}

def getBranchName(branch) {
  // special macro to look for the latest minor version
  if (branch.contains('8.<minor>')) {
   return bumpUtils.getMajorMinor(bumpUtils.getCurrentMinorReleaseFor8())
  }
  if (branch.contains('8.<next-minor>')) {
    return bumpUtils.getMajorMinor(bumpUtils.getNextMinorReleaseFor8())
  }
  if (branch.contains('7.<minor>')) {
    return bumpUtils.getMajorMinor(bumpUtils.getCurrentMinorReleaseFor7())
  }
  if (branch.contains('7.<next-minor>')) {
    return bumpUtils.getMajorMinor(bumpUtils.getNextMinorReleaseFor7())
  }
  return branch
}
