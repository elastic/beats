#!/usr/bin/env groovy

library identifier: 'apm@master',
retriever: modernSCM(
  [$class: 'GitSCMSource',
  credentialsId: 'f94e9298-83ae-417e-ba91-85c279771570',
  id: '37cf2c00-2cc7-482e-8c62-7bbffef475e2',
  remote: 'git@github.com:elastic/apm-pipeline-library.git'])

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
        //TODO we need to configure the library in Jenkins to use privileged methods.
        //gitCheckout(basedir: "${BASE_DIR}")
        dir("${BASE_DIR}"){
          checkout scm
          githubEnv()
        }
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
        script {
          env.GO_VERSION = readFile("${BASE_DIR}/.go-version").trim()
          env.BUILD_FILEBEAT = isChanged(["^filebeat/.*"])
          env.BUILD_HEARTBEAT = isChanged(["^heartbeat/.*"])
          env.BUILD_AUDITBEAT = isChanged(["^auditbeat/.*"])
          env.BUILD_METRICBEAT = isChanged(["^metricbeat/.*"])
          env.BUILD_PACKETBEAT = isChanged(["^packetbeat/.*"])
          env.BUILD_WINLOGBEAT = isChanged(["^winlogbeat/.*"])
          env.BUILD_FUNCTIONBEAT = isChanged(["^x-pack/functionbeat/.*"])
          env.BUILD_JOURNALBEAT = isChanged(["^journalbeat/.*"])
          env.BUILD_GENERATOR = isChanged(["^generator/.*"])
          env.BUILD_KUBERNETES = isChanged(["^deploy/kubernetes/*"])
          env.BUILD_DOCS = isGitRegionMatch(patterns: ["^docs/.*"], comparator: 'regexp') || params.runAllStages
          env.BUILD_LIBBEAT = isGitRegionMatch(patterns: ["^Libbeat/.*"], comparator: 'regexp') || params.runAllStages
        }
      }
    }
    stage('Lint'){
      options { skipDefaultCheckout() }
      steps {
        withBeatsEnv(){
          makeTarget("Lint", "check")
        }
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
              steps {
                withBeatsEnv(){
                  makeTarget("Filebeat oss Linux", "-C filebeat testsuite")
                }
              }
            }
            stage('Filebeat x-pack'){
              steps {
                withBeatsEnv(){
                  makeTarget("Filebeat x-pack Linux", "-C x-pack/filebeat testsuite")
                }
              }
            }
            stage('Filebeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                withBeatsEnv(){
                  makeTarget("Filebeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C filebeat testsuite")
                }
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
                withBeatsEnv(){
                  makeTarget("Heartbeat oss Linux", "-C heartbeat testsuite")
                }
              }
            }
            stage('Heartbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                withBeatsEnv(){
                  makeTarget("Heartbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C heartbeat testsuite")
                }
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
                withBeatsEnv(){
                  makeTarget("Auditbeat oss Linux", "-C auditbeat testsuite")
                }
              }
            }
            stage('Auditbeat crosscompile'){
              steps {
                withBeatsEnv(){
                  makeTarget("Auditbeat oss crosscompile", "-C auditbeat crosscompile")
                }
              }
            }
            stage('Auditbeat x-pack'){
              steps {
                withBeatsEnv(){
                  makeTarget("Auditbeat x-pack Linux", "-C x-pack/auditbeat testsuite")
                }
              }
            }
            stage('Auditbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                withBeatsEnv(){
                  makeTarget("Auditbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C auditbeat testsuite")
                }
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
                withBeatsEnv(){
                  makeTarget("Libbeat oss Linux", "-C libbeat testsuite")
                }
              }
            }
            stage('Libbeat crosscompile'){
              steps {
                withBeatsEnv(){
                  makeTarget("Libbeat oss crosscompile", "-C libbeat crosscompile")
                }
              }
            }
            stage('Libbeat stress-tests'){
              steps {
                withBeatsEnv(){
                  makeTarget("Libbeat stress-tests", "STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' -C libbeat stress-tests")
                }
              }
            }
            stage('Libbeat x-pack'){
              steps {
                withBeatsEnv(){
                  makeTarget("Libbeat x-pack Linux", "-C x-pack/libbeat testsuite")
                }
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
            withBeatsEnv(){
              makeTarget("Metricbeat Unit tests", "-C metricbeat unit-tests coverage-report")
            }
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
            withBeatsEnv(){
              makeTarget("Metricbeat Integration tests", "-C metricbeat integration-tests-environment coverage-report")
            }
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
            withBeatsEnv(){
              makeTarget("Metricbeat System tests", "-C metricbeat update system-tests-environment coverage-report")
            }
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
            withBeatsEnv(){
              makeTarget("Metricbeat x-pack Linux", "-C x-pack/metricbeat testsuite")
            }
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
            withBeatsEnv(){
              makeTarget("Metricbeat oss crosscompile", "-C metricbeat crosscompile")
            }
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
            withBeatsEnv(){
              makeTarget("Metricbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C metricbeat testsuite")
            }
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
                withBeatsEnv(){
                  makeTarget("Packetbeat oss Linux", "-C packetbeat testsuite")
                }
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
                withBeatsEnv(){
                  makeTarget("Winlogbeat oss crosscompile", "-C winlogbeat crosscompile")
                }
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
                withBeatsEnv(){
                  makeTarget("Functionbeat x-pack Linux", "-C x-pack/functionbeat testsuite")
                }
              }
            }
            stage('Functionbeat Mac OS X x-pack'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                withBeatsEnv(){
                  makeTarget("Functionbeat x-pack Mac OS X", "TEST_ENVIRONMENT=0 -C x-pack/functionbeat testsuite")
                }
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
                withBeatsEnv(){
                  makeTarget("Journalbeat Linux", "-C journalbeat testsuite")
                }
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
                withBeatsEnv(){
                  makeTarget("Generators Metricbeat Linux", "-C generator/metricbeat test")
                }
              }
            }
            stage('Generators Beat Linux'){
              steps {
                withBeatsEnv(){
                  makeTarget("Generators Beat Linux", "-C generator/beat test")
                }
              }
            }
            stage('Generators Metricbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                withBeatsEnv(){
                  makeTarget("Generators Metricbeat Mac OS X", "-C generator/metricbeat test")
                }
              }
            }
            stage('Generators Beat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              steps {
                withBeatsEnv(){
                  makeTarget("Generators Beat Mac OS X", "-C generator/beat test")
                }
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
            withBeatsEnv(){
              makeTarget("Docs", "docs")
            }
          }
        }
      }
    }
  }
}

def makeTarget(context, target, clean = true){
  withGithubNotify(context: "${context}") {
    sh(label: "Make ${target}", script: "make ${target}")
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

    def envTmp = propertiesToEnv("go_env.properties")
    withEnv(envTmp){
      dir("${BASE_DIR}"){
        try {
          body()
        } finally {
          reportCoverage()
        }
      }
    }
  }
}

def propertiesToEnv(file){
  def props = readProperties(file: file)
  if(props.containsKey('PATH')){
    newPath = sh(label: 'eval path', returnStdout: true, script: "echo \"${props['PATH']}\"").trim()
    props["PATH"] = "${newPath}:${env.PATH}"
  }
  def envTmp = []
  props.each { key, value ->
      envTmp += "${key}=${value}"
  }
  return envTmp
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
    junit(allowEmptyResults: true, keepLongStdio: true, testResults: "**/TEST-*.xml")
  }
}

def isChanged(patterns){
  return (params.runAllStages
    || isGitRegionMatch(patterns: patterns, comparator: 'regexp')
    || isGitRegionMatch(patterns: ["^libbeat/.*"], comparator: 'regexp')
  )
}
