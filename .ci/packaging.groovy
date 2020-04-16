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
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
      }
    }
    stage('Build Packages'){
      matrix {
        axes {
          // axis {
          //   name 'PLATFORMS'
          //   values (
          //     '+linux/armv7',
          //     '+linux/ppc64le',
          //     '+linux/s390x',
          //     '+linux/mips64',
          //     '+darwin',
          //     '+darwin/amd64',
          //     '+windows/386',
          //     '+windows/amd64'
          //   )
          // }
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
              'x-pack/filebeat',
              'x-pack/functionbeat',
              'x-pack/heartbeat',
              'x-pack/journalbeat',
              'x-pack/metricbeat',
              'x-pack/packetbeat',
              'x-pack/winlogbeat'
            )
          }
        }
        stages {
          stage('Package'){
            agent { label 'ubuntu && immutable' }
            options { skipDefaultCheckout() }
            environment {
              HOME = "${env.WORKSPACE}"
            }
            steps {
              deleteDir()
              unstash 'source'
              release()
              publishPackages()
            }
          }
        }
      }
    }
  }
}

def release(){
  withBeatsEnv(){
    if(env.PLATFORMS == 'darwin' && params.macos){
      withMaskEnv( vars: [
          [var: "KEYCHAIN_PASS", password: getVaultSecret(secret: "secret/jenkins-ci/macos-codesign-keychain").data.password],
          [var: "KEYCHAIN", password: "/var/lib/jenkins/Library/Keychains/Elastic.keychain-db"],
          [var: "APPLE_SIGNING_ENABLED", password: "true"],
      ]){
        sh(label: "Release ${env.BEATS_FOLDER} ${env.PLATFORMS}", script: 'mage package')
      }
    } else if (env.PLATFORMS != 'darwin' && params.linux){
      sh(label: "Release ${env.BEATS_FOLDER} ${env.PLATFORMS}", script: 'mage package')
    } else {
      unstable("Release for ${env.BEATS_FOLDER} ${env.PLATFORMS} Not executed")
    }
  }
}

def publishPackages(){
  googleStorageUpload(bucket: "gs://${JOB_GCS_BUCKET}/snapshots",
    credentialsId: "${JOB_GCS_CREDENTIALS}",
    pathPrefix: "${BASE_DIR}/build/distributions/",
    pattern: "${BASE_DIR}/build/distributions/**/*",
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
    "PYTHON_ENV=${WORKSPACE}/python-env",
//    "PLATFORMS=!defaults ${env.PLATFORMS}"
    "PLATFORMS=!defaults +linux/armv7 +linux/ppc64le +linux/s390x +linux/mips64 +windows/386 +windows/amd64"
  ]) {
    deleteDir()
    unstash 'source'
    dir("${env.BASE_DIR}/${env.BEATS_FOLDER}") {
      sh(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.sh")
      sh(label: "Install Mage", script: "make mage")
      //dockerLogin(secret: "${DOCKERELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
      //sh(label: 'workaround packer cache', '.ci/packer_cache.sh')
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
