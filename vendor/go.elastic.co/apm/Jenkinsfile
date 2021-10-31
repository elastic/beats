#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'linux && immutable' }
  environment {
    REPO = 'apm-agent-go'
    BASE_DIR = "src/go.elastic.co/apm"
    NOTIFY_TO = credentials('notify-to')
    JOB_GCS_BUCKET = credentials('gcs-bucket')
    CODECOV_SECRET = 'secret/apm-team/ci/apm-agent-go-codecov'
    GO111MODULE = 'on'
    GOPATH = "${env.WORKSPACE}"
    GOPROXY = 'https://proxy.golang.org'
    HOME = "${env.WORKSPACE}"
    GITHUB_CHECK_ITS_NAME = 'Integration Tests'
    ITS_PIPELINE = 'apm-integration-tests-selector-mbp/master'
    OPBEANS_REPO = 'opbeans-go'
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  triggers {
    issueCommentTrigger('(?i).*(?:jenkins\\W+)?run\\W+(?:the\\W+)?(?:benchmark\\W+)?tests(?:\\W+please)?.*')
  }
  parameters {
    string(name: 'GO_VERSION', defaultValue: "1.14.2", description: "Go version to use.")
    booleanParam(name: 'Run_As_Master_Branch', defaultValue: false, description: 'Allow to run any steps on a PR, some steps normally only run on master branch.')
    booleanParam(name: 'test_ci', defaultValue: true, description: 'Enable test')
    booleanParam(name: 'docker_test_ci', defaultValue: true, description: 'Enable run docker tests')
    booleanParam(name: 'bench_ci', defaultValue: true, description: 'Enable benchmarks')
  }
  stages {
    stage('Initializing'){
      options { skipDefaultCheckout() }
      environment {
        GO_VERSION = "${params.GO_VERSION}"
        PATH = "${env.PATH}:${env.WORKSPACE}/bin"
      }
      stages {
        /**
         Checkout the code and stash it, to use it on other stages.
        */
        stage('Checkout') {
          options { skipDefaultCheckout() }
          steps {
            pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
            deleteDir()
            gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true, reference: '/var/lib/jenkins/.git-references/apm-agent-go.git')
            stash allowEmpty: true, name: 'source', useDefaultExcludes: false
            script {
              dir("${BASE_DIR}"){
                // Skip all the stages except docs for PR's with asciidoc and md changes only
                env.ONLY_DOCS = isGitRegionMatch(patterns: [ '.*\\.(asciidoc|md)' ], shouldMatchAll: true)
              }
            }
          }
        }
        /**
        Execute unit tests.
        */
        stage('Tests') {
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            allOf {
              expression { return env.ONLY_DOCS == "false" }
              expression { return params.test_ci }
            }
          }
          steps {
            withGithubNotify(context: 'Tests', tab: 'tests') {
              deleteDir()
              unstash 'source'
              dir("${BASE_DIR}"){
                script {
                  def go = readYaml(file: '.jenkins.yml')
                  def parallelTasks = [:]
                  go['GO_VERSION'].each{ version ->
                    parallelTasks["Go-${version}"] = generateStep(version)
                  }
                  // For the cutting edge
                  def edge = readYaml(file: '.jenkins-edge.yml')
                  edge['GO_VERSION'].each{ version ->
                    parallelTasks["Go-${version}"] = generateStepAndCatchError(version)
                  }
                  parallel(parallelTasks)
                }
              }
            }
          }
        }
        stage('Coverage') {
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            allOf {
              expression { return env.ONLY_DOCS == "false" }
              expression { return params.docker_test_ci }
            }
          }
          steps {
            withGithubNotify(context: 'Coverage') {
              deleteDir()
              unstash 'source'
              dir("${BASE_DIR}"){
                sh script: './scripts/jenkins/before_install.sh', label: 'Install dependencies'
                sh script: './scripts/jenkins/docker-test.sh', label: 'Docker tests'
              }
            }
          }
          post {
            always {
              coverageReport("${BASE_DIR}/build/coverage")
              codecov(repo: env.REPO, basedir: "${BASE_DIR}",
                flags: "-f build/coverage/coverage.cov -X search",
                secret: "${CODECOV_SECRET}")
              junit(allowEmptyResults: true,
                keepLongStdio: true,
                testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
        stage('Benchmark') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            allOf {
              anyOf {
                branch 'master'
                tag pattern: 'v\\d+\\.\\d+\\.\\d+.*', comparator: 'REGEXP'
                expression { return params.Run_As_Master_Branch }
                expression { return env.GITHUB_COMMENT?.contains('benchmark tests') }
              }
              expression { return params.bench_ci }
            }
          }
          steps {
            withGithubNotify(context: 'Benchmark', tab: 'tests') {
              deleteDir()
              unstash 'source'
              dir("${BASE_DIR}"){
                sh script: './scripts/jenkins/before_install.sh', label: 'Install dependencies'
                sh script: './scripts/jenkins/bench.sh', label: 'Benchmarking'
                sendBenchmarks(file: 'build/bench.out', index: 'benchmark-go')
              }
            }
          }
        }
      }
    }
    stage('More OS') {
      when {
        beforeAgent true
        expression { return env.ONLY_DOCS == "false" }
      }
      parallel {
        stage('Windows') {
          agent { label 'windows-2019-immutable' }
          options { skipDefaultCheckout() }
          environment {
            GOROOT = "c:\\Go"
            GOPATH = "${env.WORKSPACE}"
            PATH = "${env.PATH};${env.GOROOT}\\bin;${env.GOPATH}\\bin"
            GO_VERSION = "${params.GO_VERSION}"
          }
          steps {
            withGithubNotify(context: 'Build-Test - Windows') {
              cleanDir("${WORKSPACE}/${BASE_DIR}")
              unstash 'source'
              dir("${BASE_DIR}"){
                bat script: 'scripts/jenkins/windows/install-tools.bat', label: 'Install tools'
                bat script: 'scripts/jenkins/windows/build-test.bat', label: 'Build and test'
              }
            }
          }
          post {
            always {
              junit(allowEmptyResults: true, keepLongStdio: true, testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
        stage('OSX') {
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          environment {
            GO_VERSION = "${params.GO_VERSION}"
            PATH = "${env.PATH}:${env.WORKSPACE}/bin"
          }
          steps {
            withGithubNotify(context: 'Build-Test - OSX') {
              retry(3) {
                deleteDir()
                unstash 'source'
                dir("${BASE_DIR}"){
                  sh script: './scripts/jenkins/before_install.sh', label: 'Install dependencies'
                }
              }
              retry(3) {
                dir("${BASE_DIR}"){
                  sh script: './scripts/jenkins/build.sh', label: 'Build'
                }
              }
              dir("${BASE_DIR}"){
                sh script: './scripts/jenkins/test.sh', label: 'Test'
              }
            }
          }
          post {
            always {
              junit(allowEmptyResults: true, keepLongStdio: true, testResults: "${BASE_DIR}/build/junit-*.xml")
              deleteDir()
            }
          }
        }
      }
    }
    stage('Integration Tests') {
      agent none
      when {
        beforeAgent true
        allOf {
          expression { return env.ONLY_DOCS == "false" }
          anyOf {
            changeRequest()
            expression { return !params.Run_As_Master_Branch }
          }
        }
      }
      steps {
        build(job: env.ITS_PIPELINE, propagate: false, wait: false,
              parameters: [string(name: 'INTEGRATION_TEST', value: 'Go'),
                           string(name: 'BUILD_OPTS', value: "--go-agent-version ${env.GIT_BASE_COMMIT} --opbeans-go-agent-branch ${env.GIT_BASE_COMMIT}"),
                           string(name: 'GITHUB_CHECK_NAME', value: env.GITHUB_CHECK_ITS_NAME),
                           string(name: 'GITHUB_CHECK_REPO', value: env.REPO),
                           string(name: 'GITHUB_CHECK_SHA1', value: env.GIT_BASE_COMMIT)])
        githubNotify(context: "${env.GITHUB_CHECK_ITS_NAME}", description: "${env.GITHUB_CHECK_ITS_NAME} ...", status: 'PENDING', targetUrl: "${env.JENKINS_URL}search/?q=${env.ITS_PIPELINE.replaceAll('/','+')}")
      }
    }
    stage('Release') {
      options { skipDefaultCheckout() }
      when {
        beforeAgent true
        tag pattern: 'v\\d+\\.\\d+\\.\\d+', comparator: 'REGEXP'
      }
      stages {
        stage('Opbeans') {
          environment {
            REPO_NAME = "${OPBEANS_REPO}"
            GO_VERSION = "${params.GO_VERSION}"
          }
          steps {
            deleteDir()
            dir("${OPBEANS_REPO}"){
              git credentialsId: 'f6c7695a-671e-4f4f-a331-acdce44ff9ba',
                  url: "git@github.com:elastic/${OPBEANS_REPO}.git"
              sh script: ".ci/bump-version.sh ${env.BRANCH_NAME}", label: 'Bump version'
              // The opbeans-go pipeline will trigger a release for the master branch
              gitPush()
              // The opbeans-go pipeline will trigger a release for the release tag
              gitCreateTag(tag: "${env.BRANCH_NAME}")
            }
          }
        }
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult()
    }
  }
}

def generateStep(version){
  return {
    node('linux && immutable'){
      try {
        echo "${version}"
        withEnv(["GO_VERSION=${version}"]) {
          // Another retry in case there are any environmental issues
          // See https://issuetracker.google.com/issues/146072599 for more context
          retry(3) {
            deleteDir()
            unstash 'source'
            dir("${BASE_DIR}"){
              sh script: './scripts/jenkins/before_install.sh', label: 'Install dependencies'
            }
          }
          retry(3) {
            dir("${BASE_DIR}"){
              sh script: './scripts/jenkins/build.sh', label: 'Build'
            }
          }
          dir("${BASE_DIR}"){
            sh script: './scripts/jenkins/test.sh', label: 'Test'
          }
        }
      } catch(e){
        error(e.toString())
      } finally {
        junit(allowEmptyResults: true,
          keepLongStdio: true,
          testResults: "${BASE_DIR}/build/junit-*.xml")
      }
    }
  }
}

def generateStepAndCatchError(version){
  return {
    catchError(buildResult: 'SUCCESS', message: 'Cutting Edge Tests', stageResult: 'UNSTABLE') {
      generateStep(version)
    }
  }
}

def cleanDir(path){
  powershell label: "Clean ${path}", script: "Remove-Item -Recurse -Force ${path}"
}
