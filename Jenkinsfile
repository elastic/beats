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
  }
  triggers {
    issueCommentTrigger('(?i).*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*')
  }
  parameters {
    booleanParam(name: 'runAllStages', defaultValue: false, description: 'Allow to run all stages.')
  }
  stages {
    /**
     Checkout the code and stash it, to use it on other stages.
    */
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        gitCheckout(basedir: "${BASE_DIR}")
        script {
          env.GO_VERSION = readFile("${BASE_DIR}/.go-version").trim()
          env.BUILD_FILEBEAT = "true" //isChanged(["^filebeat/.*"])
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
        stage('Filebeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false"
            }
          }
          stages {
            stage('Filebeat oss'){
              when {expression {return false}}
              steps {
                makeTarget("Filebeat oss Linux", "-C filebeat testsuite")
              }
            }
            stage('Filebeat x-pack'){
              when {expression {return false}}
              steps {
                makeTarget("Filebeat x-pack Linux", "-C x-pack/filebeat testsuite")
              }
            }
            stage('Filebeat Mac OS X'){
              when {expression {return false}}
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                makeTarget("Filebeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C filebeat testsuite")
              }
            }
            stage('Filebeat Windows'){
              agent { label 'windows-immutable' }
              options { skipDefaultCheckout() }
              steps {
                makeTargetWin("Filebeat oss Windows", "TEST_ENVIRONMENT=0 -C filebeat testsuite")
              }
            }
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

def makeTarget(context, target, clean = true){
  withGithubNotify(context: "${context}") {
    withBeatsEnv(){
      sh(label: "Make ${target}", script: """
        eval "\$(gvm use ${GO_VERSION} --format=bash)"
        echo make ${target}
      """)
    }
  }
}

def makeTargetWin(context, target, clean = true){
  withGithubNotify(context: "${context}") {
    withBeatsEnvWin(){
      def envVars = bat(label: 'Env vars',
        script: """
          @echo off
          gvm use --format=batch ${GO_VERSION}
        """,
        returnStdout: true
      )
      echo envVars
      echo env.WORKSPACE
      echo "${WORKSPACE}"
      echo "${env.WORKSPACE}"
      bat "echo %WORKSPACE%"
      bat(label: "Make ${target}", script: """
        ${envVars}
        set GOROOT=${WORKSPACE}\\go${GO_VERSION}.windows.amd64
        set PATH=%GOROOT%\\bin;${WORKSPACE}\\bin;C:\\tools\\mingw64\\bin;%PATH%
        set GOFLAGS=-mod=vendor
        set
        make ${target}
      """)
    }
  }
}

def withBeatsEnv(Closure body){
  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "PATH=${env.WORKSPACE}/bin:${env.PATH}",
  ]){
    deleteDir()
    unstash 'source'
    sh(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.sh")
    sh(label: "Install docker-compose ${DOCKER_COMPOSE_VERSION}", script: ".ci/scripts/install-docker-compose.sh")
    dir("${BASE_DIR}"){
      try {
        body()
      } finally {
        reportCoverage()
      }
    }
  }
}

def withBeatsEnvWin(Closure body){
  def ws = "${WORKSPACE}"
  def path = "${ws}\\bin;C:\\tools\\mingw64\\bin;${PATH}"
  echo path
  echo ws
  withEnv([
    "HOME=${ws}",
    "GOPATH=${ws}",
    "PATH=${path}",
    "MAGEFILE_CACHE=${ws}\\.magefile"
  ]){
    deleteDir()
    unstash 'source'
    powershell(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.ps1")
    dir("${BASE_DIR}"){
      try {
        body()
      } finally {
        junit(allowEmptyResults: true, keepLongStdio: true, testResults: "**/TEST-*.xml")
      }
    }
  }
}

def k8sTest(versions){
  versions.each{ v ->
    stage("k8s ${v}"){
      withEnv(["K8S_VERSION=${v}"]){
        withBeatsEnv(){
          sh(label: "Install k8s", script: ".ci/scripts/kind-setup.sh")
          makeTarget("Kubernetes Kind", "KUBECONFIG=\"\$(kind get kubeconfig-path)\" -C deploy/kubernetes test", false)
          sh(label: 'Delete cluster', script: 'kind delete cluster')
        }
      }
    }
  }
}

def reportCoverage(){
  catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
    junit(allowEmptyResults: true, keepLongStdio: true, testResults: "**/TEST-*.xml")
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
