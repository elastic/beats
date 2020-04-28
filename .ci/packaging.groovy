#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu && immutable' }
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    JOB_GCS_BUCKET = 'beats-ci-artifacts'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    DOCKERELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_REGISTRY = 'docker.elastic.co'
    SNAPSHOT = "true"
  }
  options {
    timeout(time: 3, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    disableConcurrentBuilds()
  }
  triggers {
    issueCommentTrigger('(?i)^\\/packaging$')
    upstream('Beats/beats-beats-mbp/7.7')
  }
  parameters {
    booleanParam(name: 'macos', defaultValue: false, description: 'Allow macOS stages.')
    booleanParam(name: 'linux', defaultValue: true, description: 'Allow linux stages.')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}")
        setEnvVar("GO_VERSION", readFile("${BASE_DIR}/.go-version").trim())
        //stash allowEmpty: true, name: 'source', useDefaultExcludes: false
      }
    }
    stage('Build Packages'){
      matrix {
        axes {
          axis {
            name 'BEATS_FOLDER'
            values (
              'auditbeat',
              'filebeat',
              'heartbeat',
              'journalbeat',
              'metricbeat',
              'packetbeat',
              'winlogbeat',
              'x-pack/auditbeat',
              'x-pack/dockerlogbeat',
              'x-pack/filebeat',
              'x-pack/functionbeat',
              // 'x-pack/heartbeat',
              // 'x-pack/journalbeat',
              'x-pack/metricbeat',
              // 'x-pack/packetbeat',
              'x-pack/winlogbeat'
            )
          }
        }
        stages {
          stage('Package Linux'){
            agent { label 'ubuntu && immutable' }
            options { skipDefaultCheckout() }
            when {
              beforeAgent true
              expression {
                return params.linux
              }
            }
            environment {
              HOME = "${env.WORKSPACE}"
              PLATFORMS = [
                '+all',
                'linux/amd64',
                'linux/386',
                'linux/arm64',
                'linux/armv7',
                'linux/ppc64le',
                'linux/mips64',
                'linux/s390x',
                'windows/amd64',
                'windows/386',
                (params.macos ? '' : 'darwin/amd64'),
              ].join(' ')
            }
            steps {
              release()
              pushCIDockerImages()
            }
          }
          stage('Package Mac OS'){
            agent { label 'macosx' }
            options { skipDefaultCheckout() }
            when {
              beforeAgent true
              expression {
                return params.macos
              }
            }
            environment {
              HOME = "${env.WORKSPACE}"
              PLATFORMS = [
                '+all',
                'darwin/amd64',
              ].join(' ')
            }
            steps {
              withMacOSEnv(){
                release()
              }
            }
          }
        }
      }
    }
  }
}

def pushCIDockerImages(){
  sh(label: 'Push Docker image', script: '''
    if [ -n "$(command -v docker)" ]; then
      docker images || true
    fi
  ''')
}

def release(){
  withBeatsEnv(){
    dir("${env.BEATS_FOLDER}") {
      sh(label: "Release ${env.BEATS_FOLDER} ${env.PLATFORMS}", script: 'mage package')
    }
    publishPackages("${env.BEATS_FOLDER}")
  }
}

def withMacOSEnv(Closure body){
  withEnvMask( vars: [
      [var: "KEYCHAIN_PASS", password: getVaultSecret(secret: "secret/jenkins-ci/macos-codesign-keychain").data.password],
      [var: "KEYCHAIN", password: "/var/lib/jenkins/Library/Keychains/Elastic.keychain-db"],
      [var: "APPLE_SIGNING_ENABLED", password: "true"],
  ]){
    body()
  }
}

def publishPackages(baseDir){
  googleStorageUpload(bucket: "gs://${JOB_GCS_BUCKET}/snapshots",
    credentialsId: "${JOB_GCS_CREDENTIALS}",
    pathPrefix: "${baseDir}/build/distributions/",
    pattern: "${baseDir}/build/distributions/**/*",
    sharedPublicly: true,
    showInline: true
  )
}

def withBeatsEnv(Closure body) {
  def os = goos()
  def goRoot = "${env.WORKSPACE}/.gvm/versions/go${GO_VERSION}.${os}.amd64"

  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "GOROOT=${goRoot}",
    "PATH=${env.WORKSPACE}/bin:${goRoot}/bin:${env.PATH}",
    "MAGEFILE_CACHE=${WORKSPACE}/.magefile",
    "PYTHON_ENV=${WORKSPACE}/python-env"
  ]) {
    deleteDir()
    //unstash 'source'
    gitCheckout(basedir: "${BASE_DIR}")
    dir("${env.BASE_DIR}"){
      sh(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.sh")
      sh(label: "Install Mage", script: "make mage")
      body()
    }
  }
}

def goos(){
  def labels = env.NODE_LABELS

  if (labels.contains('linux')) {
    return 'linux'
  } else if (labels.contains('windows')) {
    return 'windows'
  } else if (labels.contains('darwin')) {
    return 'darwin'
  }

  error("Unhandled OS name in NODE_LABELS: " + labels)
}
