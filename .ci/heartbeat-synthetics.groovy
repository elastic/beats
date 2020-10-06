#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-20 && immutable' }
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    JOB_GCS_BUCKET = 'beats-ci-artifacts'
    JOB_GCS_BUCKET_STASH = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    DOCKERELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_REGISTRY = 'docker.elastic.co'
    SYNTHETICS = "-synthetics"
    PIPELINE_LOG_LEVEL = "INFO"
    BEATS_FOLDER = "x-pack/heartbeat"
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
    issueCommentTrigger('(?i)^\\/packag[ing|e] synthetics$')
  }
  parameters {
    booleanParam(name: 'linux', defaultValue: true, description: 'Allow linux stages.')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}")
        setEnvVar("GO_VERSION", readFile("${BASE_DIR}/.go-version").trim())
        stashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
      }
    }
    stage('Build and test'){
      steps {
        withGithubNotify(context: "Build and test") {
          withBeatsEnv{
            dir("${env.BEATS_FOLDER}") {
              sh(label: 'Build and test', script: 'mage build test')
            }
          }
        }
      }
    }
    stage('Package Linux'){
      environment {
        HOME = "${env.WORKSPACE}"
        PLATFORMS = [
          '+all',
          'linux/amd64'
        ].join(' ')
      }
      steps {
        withGithubNotify(context: "Packaging Linux ${BEATS_FOLDER}") {
          deleteDir()
          release()
          pushCIDockerImages()
        }
      }
    }
  }
}

def pushCIDockerImages(){
  catchError(buildResult: 'UNSTABLE', message: 'Unable to push Docker images', stageResult: 'FAILURE') {
    tagAndPush('heartbeat')
  }
}

def tagAndPush(name){
  def libbetaVer = sh(label: 'Get libbeat version', script: 'grep defaultBeatVersion ${BASE_DIR}/libbeat/version/version.go|cut -d "=" -f 2|tr -d \\"', returnStdout: true)?.trim()
  libbetaVer += "${env.SYNTHETICS}"

  def tagName = "${libbetaVer}"
  def oldName = "${DOCKER_REGISTRY}/beats/${name}:${libbetaVer}"
  def newName = "${DOCKER_REGISTRY}/observability-ci/${name}:${tagName}"
  def commitName = "${DOCKER_REGISTRY}/observability-ci/${name}:${env.GIT_BASE_COMMIT}"
  dockerLogin(secret: "${DOCKERELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
  retry(3){
    sh(label:'Change tag and push', script: """
      docker tag ${oldName} ${newName}
      docker push ${newName}
      docker tag ${oldName} ${commitName}
      docker push ${commitName}
    """)
  }
}

def release(){
  withBeatsEnv(){
    dir("${env.BEATS_FOLDER}") {
      sh(label: "Release ${env.BEATS_FOLDER} ${env.PLATFORMS}", script: 'mage package')
    }
    publishPackages("${env.BEATS_FOLDER}")
  }
}

def publishPackages(baseDir){
  def bucketUri = "gs://${JOB_GCS_BUCKET}/synthetics"
  def beatsFolderName = getBeatsName(baseDir)
  googleStorageUpload(bucket: "${bucketUri}/${beatsFolderName}",
    credentialsId: "${JOB_GCS_CREDENTIALS}",
    pathPrefix: "${baseDir}/build/distributions/",
    pattern: "${baseDir}/build/distributions/**/*",
    sharedPublicly: true,
    showInline: true
  )
}

/**
* There is a specific folder structure in https://staging.elastic.co/ and https://artifacts.elastic.co/downloads/
* therefore the storage bucket in GCP should follow the same folder structure.
* This is required by https://github.com/elastic/beats-tester
* e.g.
* baseDir=name -> return name
* baseDir=name1/name2/name3-> return name2
*/
def getBeatsName(baseDir) {
  return baseDir.replace('x-pack/', '')
}

def withBeatsEnv(Closure body) {
  withMageEnv(){
    withEnv([
      "PYTHON_ENV=${WORKSPACE}/python-env"
    ]) {
      unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
      dir("${env.BASE_DIR}"){
        body()
      }
    }
  }
}
