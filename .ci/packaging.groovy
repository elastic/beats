#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent none
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    JOB_GCS_BUCKET = 'beats-ci-artifacts'
    JOB_GCS_BUCKET_STASH = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    DOCKERELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_REGISTRY = 'docker.elastic.co'
    SNAPSHOT = "true"
    PIPELINE_LOG_LEVEL = "INFO"
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
    issueCommentTrigger('(?i)^\\/packag[ing|e]$')
    // disable upstream trigger on a PR basis
    upstream("Beats/beats/${ env.JOB_BASE_NAME.startsWith('PR-') ? 'none' : env.JOB_BASE_NAME }")
  }
  parameters {
    booleanParam(name: 'macos', defaultValue: false, description: 'Allow macOS stages.')
    booleanParam(name: 'linux', defaultValue: true, description: 'Allow linux stages.')
  }
  stages {
    stage('Filter build') {
      agent { label 'ubuntu && immutable' }
      when {
        beforeAgent true
        anyOf {
          triggeredBy cause: "IssueCommentCause"
          expression {
            def ret = isUserTrigger() || isUpstreamTrigger()
            if(!ret){
              currentBuild.result = 'NOT_BUILT'
              currentBuild.description = "The build has been skipped"
              currentBuild.displayName = "#${BUILD_NUMBER}-(Skipped)"
              echo("the build has been skipped due the trigger is a branch scan and the allow ones are manual, GitHub comment, and upstream job")
            }
            return ret
          }
        }
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
                  'x-pack/elastic-agent',
                  'x-pack/dockerlogbeat',
                  'x-pack/filebeat',
                  'x-pack/functionbeat',
                   'x-pack/heartbeat',
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
                  withGithubNotify(context: "Packaging Linux ${BEATS_FOLDER}") {
                    deleteDir()
                    release()
                    pushCIDockerImages()
                  }
                }
              }
              stage('Package Mac OS'){
                agent { label 'macosx-10.12' }
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
                  withGithubNotify(context: "Packaging MacOS ${BEATS_FOLDER}") {
                    deleteDir()
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
    }
  }
}

def pushCIDockerImages(){
  catchError(buildResult: 'UNSTABLE', message: 'Unable to push Docker images', stageResult: 'FAILURE') {
    if ("${env.BEATS_FOLDER}" == "auditbeat"){
      tagAndPush('auditbeat-oss')
    } else if ("${env.BEATS_FOLDER}" == "filebeat") {
      tagAndPush('filebeat-oss')
    } else if ("${env.BEATS_FOLDER}" == "heartbeat"){
      tagAndPush('heartbeat-oss')
    } else if ("${env.BEATS_FOLDER}" == "journalbeat"){
      tagAndPush('journalbeat')
      tagAndPush('journalbeat-oss')
    } else if ("${env.BEATS_FOLDER}" == "metricbeat"){
      tagAndPush('metricbeat-oss')
    } else if ("${env.BEATS_FOLDER}" == "packetbeat"){
      tagAndPush('packetbeat')
      tagAndPush('packetbeat-oss')
    } else if ("${env.BEATS_FOLDER}" == "x-pack/auditbeat"){
      tagAndPush('auditbeat')
    } else if ("${env.BEATS_FOLDER}" == "x-pack/elastic-agent") {
      tagAndPush('elastic-agent')
    } else if ("${env.BEATS_FOLDER}" == "x-pack/filebeat"){
      tagAndPush('filebeat')
    } else if ("${env.BEATS_FOLDER}" == "x-pack/heartbeat"){
        tagAndPush('heartbeat')
    } else if ("${env.BEATS_FOLDER}" == "x-pack/metricbeat"){
      tagAndPush('metricbeat')
    }
  }
}

def tagAndPush(name){
  def libbetaVer = sh(label: 'Get libbeat version', script: 'grep defaultBeatVersion ${BASE_DIR}/libbeat/version/version.go|cut -d "=" -f 2|tr -d \\"', returnStdout: true)?.trim()
  if("${env.SNAPSHOT}" == "true"){
    libbetaVer += "-SNAPSHOT"
  }

  def tagName = "${libbetaVer}"
  if (isPR()) {
    tagName = "pr-${env.CHANGE_ID}"
  }

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
  def bucketUri = "gs://${JOB_GCS_BUCKET}/snapshots"
  if (isPR()) {
    bucketUri = "gs://${JOB_GCS_BUCKET}/pull-requests/pr-${env.CHANGE_ID}"
  }
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
