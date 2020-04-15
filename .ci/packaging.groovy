#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu && immutable' }
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    JOB_GCS_BUCKET = 'beats-ci-artifacts'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
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
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
      }
    }
    stage('Linux'){
      options { skipDefaultCheckout() }
      matrix {
        agent { label 'ubuntu && immutable' }
        axes {
          axis {
            name 'PLATFORMS'
            values (
              '+linux/armv7',
              '+linux/ppc64le',
              '+linux/s390x',
              '+linux/mips64',
              'darwin',
              'windows/386',
              'windows/amd64'
            )
          }
        }
        stages {
          stage('Package'){
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
  dir("${BASE_DIR}"){
    if(env.PLATFORMS == 'darwin' && params.macos){
      withMaskEnv( vars: [
          [var: "KEYCHAIN_PASS", password: getVaultSecret(secret: "secret/jenkins-ci/macos-codesign-keychain").data.password],
          [var: "KEYCHAIN", password: "/var/lib/jenkins/Library/Keychains/Elastic.keychain-db"],
          [var: "APPLE_SIGNING_ENABLED", password: "true"],
      ]){
        sh(label: "Release ${env.PLATFORMS}", script: './dev-tools/jenkins_release.sh')
      }
    } else if (env.PLATFORMS != 'darwin' && params.linux){
      sh(label: "Release ${env.PLATFORMS}", script: './dev-tools/jenkins_release.sh')
    } else {
      unstable("Release for ${env.PLATFORMS} Not executed")
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
