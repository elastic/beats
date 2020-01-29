#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu && immutable' }
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    GOX_FLAGS = "-arch amd64"
    DOCKER_COMPOSE_VERSION = "1.21.0"
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    disableConcurrentBuilds()
//    checkoutToSubdirectory "${env.BASE_DIR}"
  }
  triggers {
    issueCommentTrigger('(?i).*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*')
  }
  parameters {
    booleanParam(name: 'runAllStages', defaultValue: false, description: 'Allow to run all stages.')
    booleanParam(name: 'windowsTest', defaultValue: false, description: 'Allow Windows stages.')
  }
  stages {
    /**
     Checkout the code and stash it, to use it on other stages.
    */
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}")
        dir("${BASE_DIR}"){
          script {
            env.GO_VERSION = readFile(".go-version").trim()
            env.BUILD_FILEBEAT = isChanged(["^filebeat/.*"])
            env.BUILD_HEARTBEAT = isChanged(["^heartbeat/.*"])
            env.BUILD_AUDITBEAT = isChanged(["^auditbeat/.*"])
            env.BUILD_METRICBEAT = isChanged(["^metricbeat/.*"])
            env.BUILD_PACKETBEAT = isChanged(["^packetbeat/.*"])
            env.BUILD_WINLOGBEAT = isChanged(["^winlogbeat/.*"])
            env.BUILD_DOCKERLOGBEAT = isChanged(["^x-pack/dockerlogbeat/.*"])
            env.BUILD_FUNCTIONBEAT = isChanged(["^x-pack/functionbeat/.*"])
            env.BUILD_JOURNALBEAT = isChanged(["^journalbeat/.*"])
            env.BUILD_GENERATOR = isChanged(["^generator/.*"])
            env.BUILD_KUBERNETES = isChanged(["^deploy/kubernetes/*"])
            env.BUILD_DOCS = isGitRegionMatch(patterns: ["^docs/.*"], comparator: 'regexp') || params.runAllStages
            env.BUILD_LIBBEAT = isGitRegionMatch(patterns: ["^Libbeat/.*"], comparator: 'regexp') || params.runAllStages
          }
        }
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
      }
    }
    stage('Lint'){
      options { skipDefaultCheckout() }
      steps {
        makeTarget("Lint", "check")
      }
    }
    stage('Build and Test'){
      failFast false
      parallel {
        stage('Filebeat oss'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false"
            }
          }
          steps {
            makeTarget("Filebeat oss Linux", "-C filebeat testsuite")
          }
        }
        stage('Filebeat x-pack'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false"
            }
          }
          steps {
            makeTarget("Filebeat x-pack Linux", "-C x-pack/filebeat testsuite")
          }
        }
        stage('Filebeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false"
            }
          }
          steps {
            makeTarget("Filebeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C filebeat testsuite")
          }
        }
        stage('Filebeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin("Filebeat oss Windows Unit test", "-d filebeat goUnitTest")
            mageTargetWin("Filebeat oss Windows Integration test", "-d filebeat goIntegTest")
          }
        }
        stage('Heartbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_HEARTBEAT != "false"
            }
          }
          stages {
            stage('Heartbeat oss'){
              steps {
                makeTarget("Heartbeat oss Linux", "-C heartbeat testsuite")
              }
            }
            stage('Heartbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                makeTarget("Heartbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C heartbeat testsuite")
              }
            }
            stage('Heartbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Heartbeat oss Windows Unit test", "-d heartbeat goTestUnit")
              }
            }
          }
        }
        stage('Auditbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT != "false"
            }
          }
          stages {
            stage('Auditbeat oss'){
              steps {
                makeTarget("Auditbeat oss Linux", "-C auditbeat testsuite")
              }
            }
            stage('Auditbeat crosscompile'){
              steps {
                makeTarget("Auditbeat oss crosscompile", "-C auditbeat crosscompile")
              }
            }
            stage('Auditbeat x-pack'){
              steps {
                makeTarget("Auditbeat x-pack Linux", "-C x-pack/auditbeat testsuite")
              }
            }
            stage('Auditbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                makeTarget("Auditbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C auditbeat testsuite")
              }
            }
            stage('Auditbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Auditbeat Windows Unit test", "-d auditbeat goUnitTest")
                mageTargetWin("Auditbeat Windows Integration test", "-d auditbeat goIntegTest")
              }
            }
          }
        }
        stage('Libbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_LIBBEAT != "false"
            }
          }
          stages {
            stage('Libbeat oss'){
              steps {
                makeTarget("Libbeat oss Linux", "-C libbeat testsuite")
              }
            }
            stage('Libbeat crosscompile'){
              steps {
                makeTarget("Libbeat oss crosscompile", "-C libbeat crosscompile")
              }
            }
            stage('Libbeat stress-tests'){
              steps {
                makeTarget("Libbeat stress-tests", "STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' -C libbeat stress-tests")
              }
            }
            stage('Libbeat x-pack'){
              steps {
                makeTarget("Libbeat x-pack Linux", "-C x-pack/libbeat testsuite")
              }
            }
          }
        }
        stage('Metricbeat Unit tests'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat Unit tests", "-C metricbeat unit-tests coverage-report")
          }
        }
        stage('Metricbeat Integration tests'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat Integration tests", "-C metricbeat integration-tests-environment coverage-report")
          }
        }
        stage('Metricbeat System tests'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat System tests", "-C metricbeat update system-tests-environment coverage-report")
          }
        }
        stage('Metricbeat oss'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat x-pack Linux", "-C x-pack/metricbeat testsuite")
          }
        }
        stage('Metricbeat crosscompile'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat oss crosscompile", "-C metricbeat crosscompile")
          }
        }
        stage('Metricbeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C metricbeat testsuite")
          }
        }
        stage('Metricbeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin("Metricbeat Windows Unit test", "-d metricbeat goUnitTest")
            mageTargetWin("Metricbeat Windows Integration test", "-d metricbeat goIntegTest")
          }
        }
        stage('Packetbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_PACKETBEAT != "false"
            }
          }
          stages {
            stage('Packetbeat oss'){
              steps {
                makeTarget("Packetbeat oss Linux", "-C packetbeat testsuite")
              }
            }
          }
        }
        stage('dockerlogbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_DOCKERLOGBEAT != "false"
            }
          }
          stages {
            stage('Dockerlogbeat'){
              steps {
                makeTarget("Elastic Log Plugin unit tests", "-C x-pack/dockerlogbeat testsuite")
              }
            }
          }
        }
        stage('Winlogbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_WINLOGBEAT != "false"
            }
          }
          stages {
            stage('Winlogbeat oss'){
              steps {
                makeTarget("Winlogbeat oss crosscompile", "-C winlogbeat crosscompile")
              }
            }
            stage('Winlogbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Winlogbeat Windows Unit test", "-d winlogbeat goUnitTest")
              }
            }
          }
        }
        stage('Functionbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FUNCTIONBEAT != "false"
            }
          }
          stages {
            stage('Functionbeat x-pack'){
              steps {
                makeTarget("Functionbeat x-pack Linux", "-C x-pack/functionbeat testsuite")
              }
            }
            stage('Functionbeat Mac OS X x-pack'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                makeTarget("Functionbeat x-pack Mac OS X", "TEST_ENVIRONMENT=0 -C x-pack/functionbeat testsuite")
              }
            }
            stage('Functionbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Functionbeat Windows Unit test", "-d x-pack/functionbeat goUnitTest")
              }
            }
          }
        }
        stage('Journalbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_JOURNALBEAT != "false"
            }
          }
          stages {
            stage('Journalbeat oss'){
              steps {
                makeTarget("Journalbeat Linux", "-C journalbeat testsuite")
              }
            }
          }
        }
        stage('Generators'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_GENERATOR != "false"
            }
          }
          stages {
            stage('Generators Metricbeat Linux'){
              steps {
                makeTarget("Generators Metricbeat Linux", "-C generator/metricbeat test")
                makeTarget("Generators Metricbeat Linux", "-C generator/metricbeat test-package")
              }
            }
            stage('Generators Beat Linux'){
              steps {
                makeTarget("Generators Beat Linux", "-C generator/beat test")
                makeTarget("Generators Beat Linux", "-C generator/beat test-package")
              }
            }
            stage('Generators Metricbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                makeTarget("Generators Metricbeat Mac OS X", "-C generator/metricbeat test")
              }
            }
            stage('Generators Beat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                makeTarget("Generators Beat Mac OS X", "-C generator/beat test")
              }
            }
          }
        }
        stage('Kubernetes'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_KUBERNETES != "false"
            }
          }
          steps {
            k8sTest(["v1.16.2","v1.15.3","v1.14.6","v1.13.10","v1.12.10","v1.11.10"])
          }
        }
        stage('Docs'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression { return env.BUILD_DOCS != "false" }
          }
          steps {
            makeTarget("Docs", "docs")
          }
        }
      }
    }
  }
}

def makeTarget(context, target){
  withGithubNotify(context: "${context}") {
    withBeatsEnv(){
      sh(label: "Make ${target}", script: """
        eval "\$(gvm use ${GO_VERSION} --format=bash)"
        make ${target}
      """)
    }
  }
}

def mageTargetWin(context, target){
  withGithubNotify(context: "${context}") {
    withBeatsEnvWin(){
      bat(label: "Mage ${target}", script: """
        set
        mage ${target}
      """)
    }
  }
}

def withBeatsEnv(Closure body){
  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "PATH+GO=${env.WORKSPACE}/bin:${env.PATH}",
    "MAGEFILE_CACHE=${WORKSPACE}\\.magefile",
    "TEST_COVERAGE=true",
    "RACE_DETECTOR=true",
  ]){
    deleteDir()
    unstash 'source'
    dir("${BASE_DIR}"){
      sh(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.sh")
      sh(label: "Install docker-compose ${DOCKER_COMPOSE_VERSION}", script: ".ci/scripts/install-docker-compose.sh")
      try {
        body()
      } finally {
        reportCoverage()
      }
    }
  }
}

def withBeatsEnvWin(Closure body){
  def goRoot = "${env.USERPROFILE}\\.gvm\\versions\\go${GO_VERSION}.windows.amd64"
  withEnv([
    "HOME=${WORKSPACE}",
    "GOPATH=${WORKSPACE}",
    "PATH+GO=${WORKSPACE}\\bin;${goRoot}\\bin;C:\\ProgramData\\chocolatey\\bin",
    "MAGEFILE_CACHE=${WORKSPACE}\\.magefile",
    "GOROOT=${goRoot}",
    "TEST_COVERAGE=true",
    "RACE_DETECTOR=true",
  ]){
    deleteDir()
    unstash 'source'
    dir("${BASE_DIR}"){
      bat(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-tools.bat")
      try {
        body()
      } finally {
        catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
          junit(allowEmptyResults: true, keepLongStdio: true, testResults: "**\\build\\TEST*.xml")
          archiveArtifacts(allowEmptyArchive: true, artifacts: '**\\build\\TEST*.out')
        }
      }
    }
  }
}

def k8sTest(versions){
  versions.each{ v ->
    stage("k8s ${v}"){
      withEnv(["K8S_VERSION=${v}"]){
        withGithubNotify(context: "K8s ${v}") {
          withBeatsEnv(){
            sh(label: "Install k8s", script: """
              eval "\$(gvm use ${GO_VERSION} --format=bash)"
              .ci/scripts/kind-setup.sh
            """)
            sh(label: "Kubernetes Kind",script: "make KUBECONFIG=\"\$(kind get kubeconfig-path)\" -C deploy/kubernetes test")
            sh(label: 'Delete cluster', script: 'kind delete cluster')
          }
        }
      }
    }
  }
}

def reportCoverage(){
  catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
    junit(allowEmptyResults: true, keepLongStdio: true, testResults: "**/TEST-*.xml")
    archiveArtifacts(allowEmptyArchive: true, artifacts: '**/TEST-*.out')
    retry(2){
      sh(label: 'Report to Codecov', script: '''
        curl -sSLo codecov https://codecov.io/bash
        for i in auditbeat filebeat heartbeat libbeat metricbeat packetbeat winlogbeat journalbeat
        do
          FILE="${i}/build/coverage/full.cov"
          if [ -f "${FILE}" ]; then
            bash codecov -f "${FILE}"
          fi
        done
      ''')
    }
  }
}

def isChanged(patterns){
  return (params.runAllStages
    || isGitRegionMatch(patterns: patterns, comparator: 'regexp')
    || isGitRegionMatch(patterns: ["^libbeat/.*"], comparator: 'regexp')
  )
}
