#!/usr/bin/env groovy

@Library('apm@current') _

import groovy.transform.Field

/**
 This is required to store the test suites we will use to trigger the E2E tests.
*/
@Field def e2eTestSuites = []

pipeline {
  agent none
  environment {
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    JOB_GCS_BUCKET = 'beats-ci-artifacts'
    JOB_GCS_BUCKET_STASH = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    JOB_GCS_EXT_CREDENTIALS = 'beats-ci-gcs-plugin-file-credentials'
    DOCKERELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_REGISTRY = 'docker.elastic.co'
    GITHUB_CHECK_E2E_TESTS_NAME = 'E2E Tests'
    SNAPSHOT = "true"
    PIPELINE_LOG_LEVEL = "INFO"
    SLACK_CHANNEL = '#beats'
    NOTIFY_TO = 'package+beats-beats-contrib@elastic.co'
  }
  options {
    timeout(time: 4, unit: 'HOURS')
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
  }
  stages {
    stage('Filter build') {
      agent { label 'ubuntu-18 && immutable' }
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
            script {
              if(isUpstreamTrigger()) {
                try {
                  copyArtifacts(filter: 'packaging.properties',
                                flatten: true,
                                projectName: "Beats/beats/${env.JOB_BASE_NAME}",
                                selector: upstream(fallbackToLastSuccessful: true))
                  def props = readProperties(file: 'packaging.properties')
                  gitCheckout(basedir: "${BASE_DIR}", branch: props.COMMIT)
                } catch(err) {
                  // Fallback to the head of the branch as used to be.
                  gitCheckout(basedir: "${BASE_DIR}")
                }
              } else {
                gitCheckout(basedir: "${BASE_DIR}")
              }
            }
            setEnvVar("GO_VERSION", readFile("${BASE_DIR}/.go-version").trim())
            // Stash without any build/dependencies context to support different architectures.
            stashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
            withMageEnv(){
              dir("${BASE_DIR}"){
                setEnvVar('BEAT_VERSION', sh(label: 'Get beat version', script: 'make get-version', returnStdout: true)?.trim())
              }
            }
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
                  'metricbeat',
                  'packetbeat',
                  'winlogbeat',
                  'x-pack/auditbeat',
                  'x-pack/dockerlogbeat',
                  'x-pack/filebeat',
                  'x-pack/functionbeat',
                   'x-pack/heartbeat',
                  'x-pack/metricbeat',
                  'x-pack/osquerybeat',
                  'x-pack/packetbeat',
                  'x-pack/winlogbeat'
                )
              }
            }
            stages {
              stage('Package Linux'){
                agent { label 'ubuntu-18 && immutable' }
                options { skipDefaultCheckout() }
                environment {
                  HOME = "${env.WORKSPACE}"
                  PLATFORMS = [
                    '+all',
                    'linux/amd64',
                    'linux/386',
                    'linux/arm64',
                    // armv7 packaging isn't working, and we don't currently
                    // need it for release. Do not re-enable it without
                    // confirming it is fixed, you will break the packaging
                    // pipeline!
                    //'linux/armv7',
                    // The platforms above are disabled temporarly as crossbuild images are
                    // not available. See: https://github.com/elastic/golang-crossbuild/issues/71
                    //'linux/ppc64le',
                    //'linux/mips64',
                    //'linux/s390x',
                    'windows/amd64',
                    'windows/386',
                    (params.macos ? '' : 'darwin/amd64'),
                  ].join(' ')
                }
                steps {
                  withGithubNotify(context: "Packaging Linux ${BEATS_FOLDER}") {
                    deleteDir()
                    release()
                    dir("${BASE_DIR}"){
                      pushCIDockerImages(arch: 'amd64')
                    }
                  }
                  prepareE2ETestForPackage("${BEATS_FOLDER}")
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
                    deleteWorkspace()
                    withMacOSEnv(){
                      release()
                    }
                  }
                }
                post {
                  always {
                    // static workers require this
                    deleteWorkspace()
                  }
                }
              }
            }
          }
        }
        stage('Build Packages ARM'){
          matrix {
            axes {
              axis {
                name 'BEATS_FOLDER'
                values (
                  'auditbeat',
                  'filebeat',
                  'heartbeat',
                  'metricbeat',
                  'packetbeat',
                  'x-pack/auditbeat',
                  'x-pack/dockerlogbeat',
                  'x-pack/filebeat',
                  'x-pack/heartbeat',
                  'x-pack/metricbeat',
                  'x-pack/packetbeat'
                )
              }
            }
            stages {
              stage('Package Docker images for linux/arm64'){
                agent { label 'arm' }
                options { skipDefaultCheckout() }
                environment {
                  HOME = "${env.WORKSPACE}"
                  PACKAGES = "docker"
                  PLATFORMS = [
                    'linux/arm64',
                  ].join(' ')
                }
                steps {
                  withGithubNotify(context: "Packaging linux/arm64 ${BEATS_FOLDER}") {
                    deleteWorkspace()
                    release()
                    dir("${BASE_DIR}"){
                      pushCIDockerImages(arch: 'arm64')
                    }
                  }
                }
                post {
                  always {
                    // static workers require this
                    deleteWorkspace()
                  }
                }
              }
            }
          }
        }
        stage('Run E2E Tests for Packages'){
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression { return false }
          }
          steps {
            runE2ETests()
          }
        }
        stage('DRA') {
          environment {
            URI_SUFFIX = "commits/${env.GIT_BASE_COMMIT}"
            PATH_PREFIX = "${env.JOB_GCS_BUCKET.contains('/') ? env.JOB_GCS_BUCKET.substring(env.JOB_GCS_BUCKET.indexOf('/') + 1) + '/' + env.URI_SUFFIX : env.URI_SUFFIX}"
            BUCKET_URI = "gs://${env.JOB_GCS_BUCKET}/${env.REPO}/${env.URI_SUFFIX}"
          }
          steps {
            dir("${BASE_DIR}") {
              // TODO: as long as googleStorageDownload does not support recursive copy with **/*
              dir("build/distributions") {
                gsutil(command: "-m -q cp -r ${env.BUCKET_URI} .", credentialsId: env.JOB_GCS_EXT_CREDENTIALS)
                sh(label: 'move one level up', script: "mv ${env.GIT_BASE_COMMIT}/** .")
              }
              dockerLogin(secret: env.DOCKERELASTIC_SECRET, registry: env.DOCKER_REGISTRY)
              script {
                getVaultSecret.readSecretWrapper {
                  sh(label: 'release-manager.sh', script: '.ci/scripts/release-manager.sh')
                }
              }
            }
          }
          post {
            failure {
              echo 'disabled'
            // notifyStatus(slackStatus: 'danger', subject: "[${env.REPO}] DRA failed", body: "Build: (<${env.RUN_DISPLAY_URL}|here>)")
            }
          }
        }
      }
      post {
        success {
          writeFile(file: 'beats-tester.properties',
                    text: """\
                    ## To be consumed by the beats-tester pipeline
                    COMMIT=${env.GIT_BASE_COMMIT}
                    BEATS_URL_BASE=https://storage.googleapis.com/${env.JOB_GCS_BUCKET}/commits/${env.GIT_BASE_COMMIT}
                    VERSION=${env.BEAT_VERSION}-SNAPSHOT""".stripIndent()) // stripIdent() requires '''/
          archiveArtifacts artifacts: 'beats-tester.properties'
        }
      }
    }
  }
}

/**
* @param arch what architecture
*/
def pushCIDockerImages(Map args = [:]) {
  def arch = args.get('arch', 'amd64')
  catchError(buildResult: 'UNSTABLE', message: 'Unable to push Docker images', stageResult: 'FAILURE') {
    def defaultVariants = [ '' : 'beats', '-oss' : 'beats', '-ubi8' : 'beats' ]
    if (env?.BEATS_FOLDER?.endsWith('auditbeat')) {
      tagAndPush(beatName: 'auditbeat', arch: arch, variants: defaultVariants)
    } else if (env?.BEATS_FOLDER?.endsWith('filebeat')) {
      tagAndPush(beatName: 'filebeat', arch: arch, variants: defaultVariants)
    } else if (env?.BEATS_FOLDER?.endsWith('heartbeat')) {
      tagAndPush(beatName: 'heartbeat', arch: arch, variants: defaultVariants)
    } else if (env?.BEATS_FOLDER?.endsWith('metricbeat')) {
      tagAndPush(beatName: 'metricbeat', arch: arch, variants: defaultVariants)
    } else if (env?.BEATS_FOLDER?.endsWith('osquerybeat')) {
      tagAndPush(beatName: 'osquerybeat', arch: arch, variants: defaultVariants)
    } else if ("${env.BEATS_FOLDER}" == "packetbeat"){
      tagAndPush(beatName: 'packetbeat', arch: arch)
    }
  }
}

/**
* @param beatName name of the Beat
* @param arch what architecture
* @param variants list of docker variants
*/
def tagAndPush(Map args = [:]) {
  def images = [ ]
  args.variants.each { variant, sourceNamespace ->
    images += [ source: "${sourceNamespace}/${args.beatName}${variant}",
                target: "observability-ci/${args.beatName}",
                arch: args.arch ]
  }
  pushDockerImages(
    registry: env.DOCKER_REGISTRY,
    secret: env.DOCKERELASTIC_SECRET,
    snapshot: env.SNAPSHOT,
    version: env.BEAT_VERSION,
    images: images
  )
}

def prepareE2ETestForPackage(String beat){
  if ("${beat}" == "filebeat" || "${beat}" == "x-pack/filebeat") {
    e2eTestSuites.push('fleet')
    e2eTestSuites.push('helm')
  } else if ("${beat}" == "metricbeat" || "${beat}" == "x-pack/metricbeat") {
    e2eTestSuites.push('ALL')
    echo("${beat} adds all test suites to the E2E tests job.")
  } else {
    echo("${beat} does not add any test suite to the E2E tests job.")
    return
  }
}

def release(){
  withBeatsEnv(){
    withEnv([
      "DEV=true"
    ]) {
      dockerLogin(secret: "${DOCKERELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
      dir("${env.BEATS_FOLDER}") {
        sh(label: "Release ${env.BEATS_FOLDER} ${env.PLATFORMS}", script: 'mage package')
        uploadPackagesToGoogleBucket(
          credentialsId: env.JOB_GCS_EXT_CREDENTIALS,
          repo: env.REPO,
          bucket: env.JOB_GCS_BUCKET,
          folder: getBeatsName(env.BEATS_FOLDER),
          pattern: "build/distributions/**/*"
        )
      }
    }
  }
}

def runE2ETests(){
  if (e2eTestSuites.size() == 0) {
    echo("Not triggering E2E tests for PR-${env.CHANGE_ID} because the changes does not affect the E2E.")
    return
  }

  def suites = '' // empty value represents all suites in the E2E tests

  catchError(buildResult: 'UNSTABLE', message: 'Unable to run e2e tests', stageResult: 'FAILURE') {
    def suitesSet = e2eTestSuites.toSet()

    if (!suitesSet.contains('ALL')) {
      suitesSet.each { suite ->
        suites += "${suite},"
      };
    }
    echo 'runE2E will run now in a sync mode to validate packages can be published.'
    runE2E(runTestsSuites: suites,
           beatVersion: "${env.BEAT_VERSION}-SNAPSHOT",
           gitHubCheckName: env.GITHUB_CHECK_E2E_TESTS_NAME,
           gitHubCheckRepo: env.REPO,
           gitHubCheckSha1: env.GIT_BASE_COMMIT,
           propagate: true,
           wait: true)
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
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
  fixPermissions()
  withMageEnv(){
    withEnv([
      "PYTHON_ENV=${WORKSPACE}/python-env"
    ]) {
      dir("${env.BASE_DIR}"){
        body()
      }
    }
  }
}

/**
* This method fixes the filesystem permissions after the build has happenend. The reason is to
* ensure any non-ephemeral workers don't have any leftovers that could cause some environmental
* issues.
*/
def deleteWorkspace() {
  catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
    fixPermissions()
    deleteDir()
  }
}

def fixPermissions() {
  if(isUnix()) {
    catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
      dir("${env.BASE_DIR}") {
        if (fileExists('script/fix_permissions.sh')) {
          sh(label: 'Fix permissions', script: """#!/usr/bin/env bash
            set +x
            source ./dev-tools/common.bash
            docker_setup
            script/fix_permissions.sh ${WORKSPACE}""", returnStatus: true)
        }
      }
    }
  }
}

def notifyStatus(def args = [:]) {
  releaseNotification(slackChannel: "${env.SLACK_CHANNEL}",
                      slackColor: args.slackStatus,
                      slackCredentialsId: 'jenkins-slack-integration-token',
                      to: "${env.NOTIFY_TO}",
                      subject: args.subject,
                      body: args.body)
}
