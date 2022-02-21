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
    issueCommentTrigger('(?i)^\\/packag[ing|e](?:\\W+macosx)?$')
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
                  'x-pack/elastic-agent',
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
                when {
                  not { expression { return env.GITHUB_COMMENT?.contains('macosx') } }
                }
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
                agent { label 'macosx' }
                options { skipDefaultCheckout() }
                when {
                  beforeAgent true
                  anyOf {
                    expression { return params.macos }
                    expression { return env.GITHUB_COMMENT?.contains('macosx') }
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
          when {
            not { expression { return env.GITHUB_COMMENT?.contains('macosx') } }
          }
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
                  'x-pack/elastic-agent',
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
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          steps {
            runE2ETests()
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
    if (env?.BEATS_FOLDER?.endsWith('auditbeat')) {
      tagAndPush(beatName: 'auditbeat', arch: arch)
    } else if (env?.BEATS_FOLDER?.endsWith('filebeat')) {
      tagAndPush(beatName: 'filebeat', arch: arch)
    } else if (env?.BEATS_FOLDER?.endsWith('heartbeat')) {
      tagAndPush(beatName: 'heartbeat', arch: arch)
    } else if (env?.BEATS_FOLDER?.endsWith('metricbeat')) {
      tagAndPush(beatName: 'metricbeat', arch: arch)
    } else if (env?.BEATS_FOLDER?.endsWith('osquerybeat')) {
      tagAndPush(beatName: 'osquerybeat', arch: arch)
    } else if ("${env.BEATS_FOLDER}" == "packetbeat"){
      tagAndPush(beatName: 'packetbeat', arch: arch)
    } else if ("${env.BEATS_FOLDER}" == "x-pack/elastic-agent") {
      tagAndPush(beatName: 'elastic-agent', arch: arch)
    }
  }
}

/**
* @param beatName name of the Beat
* @param arch what architecture
*/
def tagAndPush(Map args = [:]) {
  def beatName = args.beatName
  def arch = args.get('arch', 'amd64')
  def libbetaVer = env.BEAT_VERSION
  def aliasVersion = ""
  if("${env.SNAPSHOT}" == "true"){
    aliasVersion = libbetaVer.substring(0, libbetaVer.lastIndexOf(".")) // remove third number in version

    libbetaVer += "-SNAPSHOT"
    aliasVersion += "-SNAPSHOT"
  }

  def tagName = "${libbetaVer}"
  if (isPR()) {
    tagName = "pr-${env.CHANGE_ID}"
  }

  // supported tags
  def tags = [tagName, "${env.GIT_BASE_COMMIT}"]
  if (!isPR() && aliasVersion != "") {
    tags << aliasVersion
  }
  // supported image flavours
  def variants = ["", "-oss", "-ubi8"]

  if(beatName == 'elastic-agent'){
      variants.add("-complete")
      variants.add("-cloud")
  }

  variants.each { variant ->
    // cloud docker images are stored in the private docker namespace.
    def sourceNamespace = variant.equals('-cloud') ? 'beats-ci' : 'beats'
    tags.each { tag ->
      // TODO:
      // For backward compatibility let's ensure we tag only for amd64, then E2E can benefit from until
      // they support the versioning with the architecture
      if ("${arch}" == "amd64") {
        doTagAndPush(beatName: beatName, variant: variant, sourceTag: libbetaVer, targetTag: "${tag}", sourceNamespace: sourceNamespace)
      }
      doTagAndPush(beatName: beatName, variant: variant, sourceTag: libbetaVer, targetTag: "${tag}-${arch}", sourceNamespace: sourceNamespace)
    }
  }
}

/**
* @param beatName name of the Beat
* @param variant name of the variant used to build the docker image name
* @param sourceNamespace namespace to be used as source for the docker tag command
* @param sourceTag tag to be used as source for the docker tag command, usually under the 'beats' namespace
* @param targetTag tag to be used as target for the docker tag command, usually under the 'observability-ci' namespace
*/
def doTagAndPush(Map args = [:]) {
  def beatName = args.beatName
  def variant = args.variant
  def sourceTag = args.sourceTag
  def targetTag = args.targetTag
  def sourceNamespace = args.sourceNamespace
  def sourceName = "${DOCKER_REGISTRY}/${sourceNamespace}/${beatName}${variant}:${sourceTag}"
  def targetName = "${DOCKER_REGISTRY}/observability-ci/${beatName}${variant}:${targetTag}"
  def iterations = 0
  retryWithSleep(retries: 3, seconds: 5, backoff: true) {
    iterations++
    def status = sh(label: "Change tag and push ${targetName}",
                    script: ".ci/scripts/docker-tag-push.sh ${sourceName} ${targetName}",
                    returnStatus: true)
    if ( status > 0 && iterations < 3) {
      error("tag and push failed for ${beatName}, retry")
    } else if ( status > 0 ) {
      log(level: 'WARN', text: "${beatName} doesn't have ${variant} docker images. See https://github.com/elastic/beats/pull/21621")
    }
  }
}

def prepareE2ETestForPackage(String beat){
  if ("${beat}" == "filebeat" || "${beat}" == "x-pack/filebeat") {
    e2eTestSuites.push('fleet')
    e2eTestSuites.push('helm')
  } else if ("${beat}" == "metricbeat" || "${beat}" == "x-pack/metricbeat") {
    e2eTestSuites.push('ALL')
    echo("${beat} adds all test suites to the E2E tests job.")
  } else if ("${beat}" == "x-pack/elastic-agent") {
    e2eTestSuites.push('fleet')
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
      }
    }
    publishPackages("${env.BEATS_FOLDER}")
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

    runE2E(runTestsSuites: suites,
           beatVersion: "${env.BEAT_VERSION}-SNAPSHOT",
           gitHubCheckName: env.GITHUB_CHECK_E2E_TESTS_NAME,
           gitHubCheckRepo: env.REPO,
           gitHubCheckSha1: env.GIT_BASE_COMMIT)
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
  uploadPackages("${bucketUri}/${beatsFolderName}", baseDir)

  // Copy those files to another location with the sha commit to test them
  // afterward.
  bucketUri = "gs://${JOB_GCS_BUCKET}/commits/${env.GIT_BASE_COMMIT}"
  uploadPackages("${bucketUri}/${beatsFolderName}", baseDir)
}

def uploadPackages(bucketUri, beatsFolder){
  googleStorageUploadExt(bucket: bucketUri,
    credentialsId: "${JOB_GCS_EXT_CREDENTIALS}",
    pattern: "${beatsFolder}/build/distributions/**/*",
    sharedPublicly: true)
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
