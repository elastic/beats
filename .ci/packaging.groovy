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
    PIPELINE_LOG_LEVEL = "INFO"
    SLACK_CHANNEL = '#ingest-notifications'
    NOTIFY_TO = 'beats-contrib+package-beats@elastic.co'
    DRA_OUTPUT = 'release-manager.out'
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
    booleanParam(name: 'run_e2e', defaultValue: true, description: 'Allow to disable the e2e tets. This workaround will generate broken/buggy binaries.')
  }
  stages {
    stage('Filter build') {
      options { skipDefaultCheckout() }
      agent { label 'ubuntu-22 && immutable' }
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
      environment {
        HOME = "${env.WORKSPACE}"
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
            dir("${BASE_DIR}"){
              setEnvVar('BEAT_VERSION', sh(label: 'Get beat version', script: 'make get-version', returnStdout: true)?.trim())
            }
            setEnvVar('IS_BRANCH_AVAILABLE', isBranchUnifiedReleaseAvailable(env.BRANCH_NAME))
          }
        }
        stage('Build Packages'){
          options { skipDefaultCheckout() }
          steps {
            generateSteps()
          }
        }
        stage('Run E2E Tests for Packages'){
          options { skipDefaultCheckout() }
          when {
            expression { return params.run_e2e }
          }
          steps {
            runE2ETests()
          }
        }
        stage('DRA Snapshot') {
          options { skipDefaultCheckout() }
          // The Unified Release process keeps moving branches as soon as a new
          // minor version is created, therefore old release branches won't be able
          // to use the release manager as their definition is removed.
          when {
            expression { return env.IS_BRANCH_AVAILABLE == "true" }
          }
          steps {
            runReleaseManager(type: 'snapshot', outputFile: env.DRA_OUTPUT)
          }
          post {
            failure {
              notifyStatus(analyse: true,
                           file: "${BASE_DIR}/${env.DRA_OUTPUT}",
                           subject: "[${env.REPO}@${env.BRANCH_NAME}] The Daily releasable artifact failed.",
                           body: 'Contact the Release Platform team [#platform-release]')
            }
          }
        }
        stage('DRA Release Staging') {
          options { skipDefaultCheckout() }
          when {
            allOf {
              // The Unified Release process keeps moving branches as soon as a new
              // minor version is created, therefore old release branches won't be able
              // to use the release manager as their definition is removed.
              expression { return env.IS_BRANCH_AVAILABLE == "true" }
              not { branch 'main' }
            }
          }
          steps {
            runReleaseManager(type: 'staging', outputFile: env.DRA_OUTPUT)
          }
          post {
            failure {
              notifyStatus(analyse: true,
                           file: "${BASE_DIR}/${env.DRA_OUTPUT}",
                           subject: "[${env.REPO}@${env.BRANCH_NAME}] The Daily releasable artifact failed.",
                           body: 'Contact the Release Platform team [#platform-release]')
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
                    BEATS_URL_BASE=https://storage.googleapis.com/${env.JOB_GCS_BUCKET}/${env.REPO}/commits/${env.GIT_BASE_COMMIT}
                    VERSION=${env.BEAT_VERSION}-SNAPSHOT""".stripIndent()) // stripIdent() requires '''/
          archiveArtifacts artifacts: 'beats-tester.properties'
        }
      }
    }
  }
}

def getBucketUri(type) {
  // It uses the folder structure done in uploadPackagesToGoogleBucket
  // commit for the normal workflow, snapshots (aka SNAPSHOT=true)
  // staging for the staging workflow, SNAPSHOT=false
  def folder = type.equals('staging') ? 'staging' : 'commits'
  return "gs://${env.JOB_GCS_BUCKET}/${env.REPO}/${folder}/${env.GIT_BASE_COMMIT}"
}

def runReleaseManager(def args = [:]) {
  def type = args.get('type', 'snapshot')
  def bucketUri = getBucketUri(type)
  deleteDir()
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
  dir("${BASE_DIR}") {
    unstash "dependencies-${type}"
    // TODO: as long as googleStorageDownload does not support recursive copy with **/*
    dir("build/distributions") {
      gsutil(command: "-m -q cp -r ${bucketUri} .", credentialsId: env.JOB_GCS_EXT_CREDENTIALS)
      sh(label: 'move one level up', script: "mv ${env.GIT_BASE_COMMIT}/** .")
    }
    sh(label: "prepare-release-manager-artifacts ${type}", script: ".ci/scripts/prepare-release-manager.sh ${type}")
    dockerLogin(secret: env.DOCKERELASTIC_SECRET, registry: env.DOCKER_REGISTRY)
    releaseManager(project: 'beats',
                   version: env.BEAT_VERSION,
                   type: type,
                   artifactsFolder: 'build/distributions',
                   outputFile: args.outputFile)
  }
}

def generateSteps() {
  def parallelTasks = [:]
  def beats = [
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
  ]

  def armBeats = [
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
  ]
  beats.each { beat ->
    parallelTasks["linux-${beat}"] = generateLinuxStep(beat)
    if (armBeats.contains(beat)) {
      parallelTasks["arm-${beat}"] =  generateArmStep(beat)
    }
  }

  // enable beats-dashboards within the existing worker
  parallelTasks["beats-dashboards"] = {
    withGithubNotify(context: "beats-dashboards") {
      withEnv(["HOME=${env.WORKSPACE}"]) {
        ['snapshot', 'staging'].each { type ->
          deleteDir()
          withBeatsEnv(type) {
            sh(label: 'make dependencies.csv', script: 'make build/distributions/dependencies.csv')
            sh(label: 'make beats-dashboards', script: 'make beats-dashboards')
            stash(includes: 'build/distributions/**', name: "dependencies-${type}", useDefaultExcludes: false)
          }
        }
      }
    }
  }
  parallel(parallelTasks)
}

def generateArmStep(beat) {
  return {
    withNode(labels: 'ubuntu-2204-aarch64') {
      withEnv(["HOME=${env.WORKSPACE}", 'PLATFORMS=linux/arm64','PACKAGES=docker', "BEATS_FOLDER=${beat}"]) {
        withGithubNotify(context: "Packaging Arm ${beat}") {
          deleteDir()
          release('snapshot')
          dir("${BASE_DIR}"){
            pushCIDockerImages(arch: 'arm64')
          }
        }
        // Staging is only needed from branches (main or release branches)
        if (isBranch()) {
          deleteDir()
          release('staging')
        }
      }
    }
  }
}

def generateLinuxStep(beat) {
  return {
    withNode(labels: 'ubuntu-22.04 && immutable') {
      withEnv(["HOME=${env.WORKSPACE}", "PLATFORMS=${linuxPlatforms()}", "BEATS_FOLDER=${beat}"]) {
        withGithubNotify(context: "Packaging Linux ${beat}") {
          deleteDir()
          release('snapshot')
          dir("${BASE_DIR}"){
            pushCIDockerImages(arch: 'amd64')
          }
        }
        prepareE2ETestForPackage("${beat}")

        // Staging is only needed from branches (main or release branches)
        if (isBranch()) {
          // As long as we reuse the same worker to package more than
          // once, the workspace gets corrupted with some permissions
          // therefore let's reset the workspace to a new location
          // in order to reuse the worker and successfully run the package
          def work = "workspace/${env.JOB_BASE_NAME}-${env.BUILD_NUMBER}-staging"
          ws(work) {
            withEnv(["HOME=${env.WORKSPACE}"]) {
              deleteDir()
              release('staging')
            }
          }
        }
      }
    }
  }
}

def linuxPlatforms() {
  return [
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
            'darwin/amd64',
            'darwin/arm64'
          ].join(' ')
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
    snapshot: true,
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

def release(type){
  withBeatsEnv(type){
    // As agreed DEV=false for staging otherwise DEV=true
    // this should avoid releasing binaries with the debug symbols and disabled most build optimizations.
    withEnv([
      "DEV=${!type.equals('staging')}"
    ]) {
      dockerLogin(secret: "${DOCKERELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
      dir("${env.BEATS_FOLDER}") {
        sh(label: "mage package ${type} ${env.BEATS_FOLDER} ${env.PLATFORMS}", script: 'mage package')
        sh(label: "mage ironbank ${type} ${env.BEATS_FOLDER} ${env.PLATFORMS}", script: 'mage ironbank')
        def folder = getBeatsName(env.BEATS_FOLDER)
        uploadPackagesToGoogleBucket(
          credentialsId: env.JOB_GCS_EXT_CREDENTIALS,
          repo: env.REPO,
          bucket: env.JOB_GCS_BUCKET,
          folder: folder,
          pattern: "build/distributions/*"
        )
        if (type.equals('staging')) {
          dir("build/distributions") {
            def bucketUri = getBucketUri(type)
            log(level: 'INFO', text: "${env.BEATS_FOLDER} for ${type} requires to upload the artifacts to ${bucketUri}.")
            googleStorageUploadExt(bucket: "${bucketUri}/${folder}",
                                   credentialsId: "${JOB_GCS_EXT_CREDENTIALS}",
                                   pattern: "*",
                                   sharedPublicly: true)
          }
        } else {
          log(level: 'INFO', text: "${env.BEATS_FOLDER} for ${type} does not require to upload the staging artifacts.")
        }
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

  def suitesSet = e2eTestSuites.toSet()

  if (!suitesSet.contains('ALL')) {
    suitesSet.each { suite ->
      suites += "${suite},"
    };
  }
  echo 'runE2E has asynchronously triggered the end to end test job.'
  runE2E(runTestsSuites: suites,
        testMatrixFile: '.ci/.e2e-tests-beats.yaml',
        beatVersion: "${env.BEAT_VERSION}-SNAPSHOT",
        gitHubCheckName: env.GITHUB_CHECK_E2E_TESTS_NAME,
        gitHubCheckRepo: env.REPO,
        gitHubCheckSha1: env.GIT_BASE_COMMIT,
        propagate: false, // Ignore the result of the downstream E2E job.
        wait: false) // Do not synchronously wait for the downstream E2E job to complete.
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

def withBeatsEnv(type, Closure body) {
  def envVars = [ "PYTHON_ENV=${WORKSPACE}/python-env" ]
  if (type.equals('snapshot')) {
    envVars << "SNAPSHOT=true"
  }
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET_STASH}", credentialsId: "${JOB_GCS_CREDENTIALS}")
  withMageEnv(){
    withEnv(envVars) {
      dir("${env.BASE_DIR}"){
        body()
      }
    }
  }
}

def notifyStatus(def args = [:]) {
  def releaseManagerFile = args.get('file', '')
  def analyse = args.get('analyse', false)
  def subject = args.get('subject', '')
  def body = args.get('body', '')
  releaseManagerNotification(file: releaseManagerFile,
                             analyse: analyse,
                             slackChannel: "${env.SLACK_CHANNEL}",
                             slackColor: 'danger',
                             slackCredentialsId: 'jenkins-slack-integration-token',
                             to: "${env.NOTIFY_TO}",
                             subject: subject,
                             body: "Build: (<${env.RUN_DISPLAY_URL}|here>).\n ${body}")
}
