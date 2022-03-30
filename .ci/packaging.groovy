#!/usr/bin/env groovy

@Library('apm@test/releaseManager') _

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
    NOTIFY_TO = 'beats-contrib+package-beats@elastic.co'
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
  stages {
    stage('Filter build') {
      options { skipDefaultCheckout() }
      agent { label 'ubuntu-20 && immutable' }
      when {
        beforeAgent true
        anyOf {
          triggeredBy cause: "IssueCommentCause"
          expression {
            // TODO: for testing purposes
            return true
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
            //TODO: for testing purposes
            //setEnvVar('IS_BRANCH_AVAILABLE', isBranchUnifiedReleaseAvailable(env.BRANCH_NAME))
            setEnvVar('IS_BRANCH_AVAILABLE', isBranchUnifiedReleaseAvailable('main'))
          }
        }
        stage('Build Packages'){
          options { skipDefaultCheckout() }
          steps {
            generateSteps()
          }
        }
        stage('Run E2E Tests for Packages'){
          // TODO: for testing purposes
          when { expression { return false } }
          options { skipDefaultCheckout() }
          steps {
            runE2ETests()
          }
        }
        stage('DRA') {
          options { skipDefaultCheckout() }
          // The Unified Release process keeps moving branches as soon as a new
          // minor version is created, therefore old release branches won't be able
          // to use the release manager as their definition is removed.
          when {
            expression { return env.IS_BRANCH_AVAILABLE == "true" }
          }
          environment {
            // It uses the folder structure done in uploadPackagesToGoogleBucket
            BUCKET_URI = "gs://${env.JOB_GCS_BUCKET}/${env.REPO}/commits/${env.GIT_BASE_COMMIT}"
            DRA_OUTPUT = 'release-manager.out'
          }
          steps {
            dir("${BASE_DIR}") {
              // TODO: as long as googleStorageDownload does not support recursive copy with **/*
              dir("build/distributions") {
                gsutil(command: "-m -q cp -r ${env.BUCKET_URI} .", credentialsId: env.JOB_GCS_EXT_CREDENTIALS)
                sh(label: 'move one level up', script: "mv ${env.GIT_BASE_COMMIT}/** .")
              }
              sh(label: "debug package", script: 'find build/distributions -type f -ls || true')
              sh(label: 'prepare-release-manager-artifacts', script: ".ci/scripts/prepare-release-manager.sh")
              dockerLogin(secret: env.DOCKERELASTIC_SECRET, registry: env.DOCKER_REGISTRY)
              //TODO: for testing purposes
              withEnv(["BRANCH_NAME=main"]){
              releaseManager(project: 'beats',
                             version: env.BEAT_VERSION,
                             type: 'snapshot',
                             artifactsFolder: 'build/distributions',
                             outputFile: env.DRA_OUTPUT)
              }
            }
          }
          post {
            failure {
              notifyStatus(analyse: true,
                           file: "${BASE_DIR}/${env.DRA_OUTPUT}",
                           subject: "[${env.REPO}@${env.BRANCH_NAME}] DRA failed.",
                           body: 'Contact the Release Platform team [#platform-release].')
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
        withBeatsEnv() {
          sh(label: 'make dependencies.csv', script: 'make build/distributions/dependencies.csv')
          sh(label: 'make beats-dashboards', script: 'make beats-dashboards')
        }
      }
    }
  }
  parallel(parallelTasks)
}

def generateArmStep(beat) {
  return {
    withNode(labels: 'arm') {
      withEnv(["HOME=${env.WORKSPACE}", 'PLATFORMS=linux/arm64','PACKAGES=docker', "BEATS_FOLDER=${beat}"]) {
        withGithubNotify(context: "Packaging Arm ${beat}") {
          deleteDir()
          release()
          dir("${BASE_DIR}"){
            pushCIDockerImages(arch: 'arm64')
          }
        }
      }
    }
  }
}

def generateLinuxStep(beat) {
  return {
    withNode(labels: 'ubuntu-20.04 && immutable') {
      withEnv(["HOME=${env.WORKSPACE}", "PLATFORMS=${linuxPlatforms()}", "BEATS_FOLDER=${beat}"]) {
        withGithubNotify(context: "Packaging Linux ${beat}") {
          deleteDir()
          release()
          dir("${BASE_DIR}"){
            pushCIDockerImages(arch: 'amd64')
          }
        }
        prepareE2ETestForPackage("${beat}")
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
        def folder = getBeatsName(env.BEATS_FOLDER)
        uploadPackagesToGoogleBucket(
          credentialsId: env.JOB_GCS_EXT_CREDENTIALS,
          repo: env.REPO,
          bucket: env.JOB_GCS_BUCKET,
          folder: folder,
          pattern: "build/distributions/*"
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

def notifyStatus(def args = [:]) {
  releaseNotification(slackChannel: "${env.SLACK_CHANNEL}",
                      slackColor: args.slackStatus,
                      slackCredentialsId: 'jenkins-slack-integration-token',
                      to: "${env.NOTIFY_TO}",
                      subject: args.subject,
                      body: args.body)
}
