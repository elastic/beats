#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu && immutable' }
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    JOB_GCS_BUCKET = 'beats-ci-artifacts'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
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
      agent { label 'ubuntu && immutable' }
      options { skipDefaultCheckout() }
      when {
        beforeAgent true
        expression { return params.linux }
      }
      environment {
        SNAPSHOT = "true"
        PLATFORMS = "+linux/armv7 +linux/ppc64le +linux/s390x +linux/mips64"
        HOME = "${env.WORKSPACE}"
      }
      steps {
        deleteDir()
        unstash 'source'
        dir("${BASE_DIR}"){
          sh(label: 'Release', script: './dev-tools/jenkins_release.sh')
        }
        publishPackages()
      }
    }
    stage('Mac OS'){
      agent { label 'macosx' }
      options { skipDefaultCheckout() }
      when {
        beforeAgent true
        expression { return params.linux }
      }
      environment {
        KEYCHAIN = "/var/lib/jenkins/Library/Keychains/Elastic.keychain-db"
        SNAPSHOT = "true"
        PLATFORMS = "darwin"
        APPLE_SIGNING_ENABLED = "true"
        HOME = "${env.WORKSPACE}"
      }
      steps {
        deleteDir()
        unstash 'source'
        withMaskEnv( vars: [
            [var: "KEYCHAIN_PASS", password: getVaultSecret(secret: "secret/jenkins-ci/macos-codesign-keychain").data.password],
        ]){
          dir("${BASE_DIR}"){
            sh(label: 'Release', script: './dev-tools/jenkins_release.sh')
          }
          publishPackages()
        }
      }
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
