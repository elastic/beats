#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu-18 && immutable' }
  environment {
    AWS_ACCOUNT_SECRET = 'secret/observability-team/ci/elastic-observability-aws-account-auth'
    AWS_REGION = "${params.awsRegion}"
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    DOCKERHUB_SECRET = 'secret/observability-team/ci/elastic-observability-dockerhub'
    DOCKER_ELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_COMPOSE_VERSION = "1.21.0"
    DOCKER_REGISTRY = 'docker.elastic.co'
    JOB_GCS_BUCKET = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    JOB_GCS_EXT_CREDENTIALS = 'beats-ci-gcs-plugin-file-credentials'
    OSS_MODULE_PATTERN = '^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
    PIPELINE_LOG_LEVEL = 'INFO'
    PYTEST_ADDOPTS = "${params.PYTEST_ADDOPTS}"
    RUNBLD_DISABLE_NOTIFICATIONS = 'true'
    SLACK_CHANNEL = "#beats-build"
    SNAPSHOT = 'true'
    TERRAFORM_VERSION = "0.12.30"
    XPACK_MODULE_PATTERN = '^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
  }
  options {
    timeout(time: 4, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '60', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
  }
  triggers {
    issueCommentTrigger('(?i)(.*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*|^/test(?:\\W+.*)?$)')
  }
  parameters {
    booleanParam(name: 'allCloudTests', defaultValue: false, description: 'Run all cloud integration tests.')
    booleanParam(name: 'awsCloudTests', defaultValue: false, description: 'Run AWS cloud integration tests.')
    string(name: 'awsRegion', defaultValue: 'eu-central-1', description: 'Default AWS region to use for testing.')
    booleanParam(name: 'runAllStages', defaultValue: false, description: 'Allow to run all stages.')
    booleanParam(name: 'armTest', defaultValue: false, description: 'Allow ARM stages.')
    booleanParam(name: 'macosTest', defaultValue: false, description: 'Allow macOS stages.')
    string(name: 'PYTEST_ADDOPTS', defaultValue: '', description: 'Additional options to pass to pytest. Use PYTEST_ADDOPTS="-k pattern" to only run tests matching the specified pattern. For retries you can use `--reruns 3 --reruns-delay 15`')
  }
  stages {
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
        deleteDir()
        // Here we do a checkout into a temporary directory in order to have the
        // side-effect of setting up the git environment correctly.
        gitCheckout(basedir: "${pwd(tmp: true)}", githubNotifyFirstTimeContributor: true)
        dir("${BASE_DIR}") {
            // We use a raw checkout to avoid the many extra objects which are brought in
            // with a `git fetch` as would happen if we used the `gitCheckout` step.
            checkout scm
        }
        stashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
        dir("${BASE_DIR}"){
          // Skip all the stages except docs for PR's with asciidoc, md or deploy k8s templates changes only
          setEnvVar('ONLY_DOCS', isGitRegionMatch(patterns: [ '(.*\\.(asciidoc|md)|deploy/kubernetes/.*-kubernetes\\.yaml)' ], shouldMatchAll: true).toString())
          setEnvVar('GO_MOD_CHANGES', isGitRegionMatch(patterns: [ '^go.mod' ], shouldMatchAll: false).toString())
          setEnvVar('PACKAGING_CHANGES', isGitRegionMatch(patterns: [ '^dev-tools/packaging/.*' ], shouldMatchAll: false).toString())
          setEnvVar('GO_VERSION', readFile(".go-version").trim())
          withEnv(["HOME=${env.WORKSPACE}"]) {
            retryWithSleep(retries: 2, seconds: 5){ sh(label: "Install Go ${env.GO_VERSION}", script: '.ci/scripts/install-go.sh') }
          }
        }
        withMageEnv(version: "${env.GO_VERSION}"){
          dir("${BASE_DIR}"){
            setEnvVar('VERSION', sh(label: 'Get beat version', script: 'make get-version', returnStdout: true)?.trim())
          }
        }
      }
    }
    stage('windows-10-20') {
      options { skipDefaultCheckout() }
      steps {
        runBuildAndTest(number: 20, context: 'windows-10-20')
      }
    }
    stage('windows-10-40') {
      options { skipDefaultCheckout() }
      steps {
        runBuildAndTest(number: 40, context: 'windows-10-40')
      }
    }
    stage('windows-10-80') {
      options { skipDefaultCheckout() }
      steps {
        runBuildAndTest(number: 80, context: 'windows-10-80')
      }
    }

  }
  post {
    cleanup {
      // Required to enable the flaky test reporting with GitHub. Workspace exists since the post/always runs earlier
      dir("${BASE_DIR}"){
        notifyBuildResult(prComment: true,
                          slackComment: true, slackNotify: (isBranch() || isTag()),
                          analyzeFlakey: !isTag(), jobName: getFlakyJobName(withBranch: getFlakyBranch()))
      }
    }
  }
}

def runBuildAndTest(Map args = [:]) {
  deleteDir()
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
  dir("${BASE_DIR}"){
    def mapParallelTasks = [:]
    for(int k = 0;k<args.number;k++) {
      mapParallelTasks["${k}"] = target(command: 'mage build',
                                        context: args.context + '-'+k,
                                        directory: 'auditbeat',
                                        isMage: true,
                                        enableRetry: false)
    }
    parallel(mapParallelTasks)
  }
}

/**
* There are only two supported branches, master and 7.x
*/
def getFlakyBranch() {
  if(isPR()) {
    return getBranchIndice(env.CHANGE_TARGET)
  } else {
    return getBranchIndice(env.BRANCH_NAME)
  }
}

/**
* There are only two supported branches, master and 7.x
*/
def getBranchIndice(String compare) {
  if (compare?.equals('master') || compare.equals('7.x')) {
    return compare
  } else {
    if (compare.startsWith('7.')) {
      return '7.x'
    }
  }
  return 'master'
}

/**
* This method runs the given command supporting two kind of scenarios:
*  - make -C <folder> then the dir(location) is not required, aka by disaling isMage: false
*  - mage then the dir(location) is required, aka by enabling isMage: true.
*/
def target(Map args = [:]) {
  def command = args.command
  def context = args.context
  def directory = args.get('directory', '')
  def withModule = args.get('withModule', false)
  def isMage = args.get('isMage', false)
  def isE2E = args.e2e?.get('enabled', false)
  def isPackaging = args.get('package', false)
  def dockerArch = args.get('dockerArch', 'amd64')
  def enableRetry = args.get('enableRetry', false)
  withNode(labels: args.label, forceWorkspace: true){
    withGithubNotify(context: "${context}") {
      withBeatsEnv(archive: true, withModule: withModule, directory: directory, id: args.id) {
        dumpVariables()
        // make commands use -C <folder> while mage commands require the dir(folder)
        // let's support this scenario with the location variable.
        dir(isMage ? directory : '') {
          if (enableRetry) {
            // Retry the same command to bypass any kind of flakiness.
            // Downside: genuine failures will be repeated.
            retry(3) {
              cmd(label: "${args.id?.trim() ? args.id : env.STAGE_NAME} - ${command}", script: "${command}")
            }
          } else {
            cmd(label: "${args.id?.trim() ? args.id : env.STAGE_NAME} - ${command}", script: "${command}")
          }
        }
        // TODO:
        // Packaging should happen only after the e2e?
        if (isPackaging) {
          publishPackages("${directory}")
        }
        if(isE2E) {
          e2e(args)
        }
        // TODO:
        // push docker images should happen only after the e2e?
        if (isPackaging) {
          pushCIDockerImages(beatsFolder: "${directory}", arch: dockerArch)
        }
      }
    }
  }
}

/**
* This method wraps all the environment setup and pre-requirements to run any commands.
*/
def withBeatsEnv(Map args = [:], Closure body) {
  def archive = args.get('archive', true)
  def withModule = args.get('withModule', false)
  def directory = args.get('directory', '')

  def path, magefile, pythonEnv, testResults, artifacts, gox_flags, userProfile

  if(isUnix()) {
    gox_flags = (isArm() && is64arm()) ? '-arch arm' : '-arch amd64'
    path = "${env.WORKSPACE}/bin:${env.PATH}"
    magefile = "${WORKSPACE}/.magefile"
    pythonEnv = "${WORKSPACE}/python-env"
    testResults = '**/build/TEST*.xml'
    artifacts = '**/build/TEST*.out'
  } else {
    // NOTE: to support Windows 7 32 bits the arch in the mingw and go context paths is required.
    def mingwArch = is32() ? '32' : '64'
    def chocoPath = 'C:\\ProgramData\\chocolatey\\bin'
    def chocoPython3Path = 'C:\\Python38;C:\\Python38\\Scripts'
    userProfile="${env.WORKSPACE}"
    path = "${env.WORKSPACE}\\bin;${chocoPath};${chocoPython3Path};C:\\tools\\mingw${mingwArch}\\bin;${env.PATH}"
    magefile = "${env.WORKSPACE}\\.magefile"
    testResults = "**\\build\\TEST*.xml"
    artifacts = "**\\build\\TEST*.out"
    gox_flags = '-arch 386'
  }

  // IMPORTANT: Somehow windows workers got a different opinion regarding removing the workspace.
  //            Windows workers are ephemerals, so this should not really affect us.
  if(isUnix()) {
    deleteDir()
  }

  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
  // NOTE: This is required to run after the unstash
  def module = withModule ? getCommonModuleInTheChangeSet(directory) : ''
  withEnv([
    "DOCKER_PULL=0",
    "GOPATH=${env.WORKSPACE}",
    "GOX_FLAGS=${gox_flags}",
    "HOME=${env.WORKSPACE}",
    "MAGEFILE_CACHE=${magefile}",
    "MODULE=${module}",
    "PATH=${path}",
    "PYTHON_ENV=${pythonEnv}",
    "RACE_DETECTOR=true",
    "TEST_COVERAGE=true",
    "TEST_TAGS=${env.TEST_TAGS},oracle",
    "OLD_USERPROFILE=${env.USERPROFILE}",
    "USERPROFILE=${userProfile}"
  ]) {
    if(isDockerInstalled()) {
      dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
      dockerLogin(secret: "${DOCKERHUB_SECRET}", registry: 'docker.io')
    }
    withMageEnv(version: "${env.GO_VERSION}"){
      dir("${env.BASE_DIR}") {
        // Go/Mage installation is not anymore configured with env variables and installed
        // with installTools but delegated to the parent closure withMageEnv.
        installTools(args)
        // Skip to upload the generated files by default.
        def upload = false
        try {
          // Add more stability when dependencies are not accessible temporarily
          // See https://github.com/elastic/beats/issues/21609
          // retry/try/catch approach reports errors, let's avoid it to keep the
          // notifications cleaner.
          if (cmd(label: 'Download modules to local cache', script: 'go mod download', returnStatus: true) > 0) {
            cmd(label: 'Download modules to local cache - retry', script: 'go mod download', returnStatus: true)
          }
          body()
        } catch(err) {
          // Upload the generated files ONLY if the step failed. This will avoid any overhead with Google Storage
          upload = true
          error("Error '${err.toString()}'")
        } finally {
          if (archive) {
            archiveTestOutput(testResults: testResults, artifacts: artifacts, id: args.id, upload: upload)
          }
          tearDown()
        }
      }
    }
  }
}

/**
* Tear down the setup for the permanent workers.
*/
def tearDown() {
  catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
    cmd(label: 'Remove the entire module cache', script: 'go clean -modcache', returnStatus: true)
    fixPermissions("${WORKSPACE}")
    // IMPORTANT: Somehow windows workers got a different opinion regarding removing the workspace.
    //            Windows workers are ephemerals, so this should not really affect us.
    if (isUnix()) {
      dir("${WORKSPACE}") {
        deleteDir()
      }
    }
  }
}

/**
* This method fixes the filesystem permissions after the build has happenend. The reason is to
* ensure any non-ephemeral workers don't have any leftovers that could cause some environmental
* issues.
*/
def fixPermissions(location) {
  if(isUnix()) {
    catchError(message: 'There were some failures when fixing the permissions', buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
      timeout(5) {
        sh(label: 'Fix permissions', script: """#!/usr/bin/env bash
          set +x
          echo "Cleaning up ${location}"
          source ./dev-tools/common.bash
          docker_setup
          script/fix_permissions.sh ${location}""", returnStatus: true)
      }
    }
  }
}

/**
* This method installs the required dependencies that are for some reason not available in the
* CI Workers.
*/
def installTools(args) {
  def stepHeader = "${args.id?.trim() ? args.id : env.STAGE_NAME}"
  if(isUnix()) {
    retryWithSleep(retries: 2, seconds: 5, backoff: true){ sh(label: "${stepHeader} - Install Python/Docker/Terraform", script: '.ci/scripts/install-tools.sh') }
    // TODO (2020-04-07): This is a work-around to fix the Beat generator tests.
    // See https://github.com/elastic/beats/issues/17787.
    sh(label: 'check git config', script: '''
      if [ -z "$(git config --get user.email)" ]; then
        git config --global user.email "beatsmachine@users.noreply.github.com"
        git config --global user.name "beatsmachine"
      fi''')
  } else {
    retryWithSleep(retries: 3, seconds: 5, backoff: true){ bat(label: "${stepHeader} - Install Python", script: ".ci/scripts/install-tools.bat") }
  }
}

/**
* This method gathers the module name, if required, in order to run the ITs only if
* the changeset affects a specific module.
*
* For such, it's required to look for changes under the module folder and exclude anything else
* such as asciidoc and png files.
*/
def getCommonModuleInTheChangeSet(String directory) {
  // Use contains to support the target(target: 'make -C <folder>') while target(directory: '<folder>', target: '...')
  def pattern = (directory.contains('x-pack') ? env.XPACK_MODULE_PATTERN : env.OSS_MODULE_PATTERN)
  def module = ''

  // Transform folder structure in regex format since path separator is required to be escaped
  def transformedDirectory = directory.replaceAll('/', '\\/')
  def directoryExclussion = "((?!^${transformedDirectory}\\/).)*\$"
  def exclude = "^(${directoryExclussion}|((?!\\/module\\/).)*\$|.*\\.asciidoc|.*\\.png)"
  dir("${env.BASE_DIR}") {
    module = getGitMatchingGroup(pattern: pattern, exclude: exclude)
    if(!fileExists("${directory}/module/${module}")) {
      module = ''
    }
  }
  return module
}

/**
* This method archives and report the tests output, for such, it searches in certain folders
* to bypass some issues when working with big repositories.
*/
def archiveTestOutput(Map args = [:]) {
  catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
    if (isUnix()) {
      fixPermissions("${WORKSPACE}")
    }
    // Remove pycache directory and go vendors cache folders
    if (isUnix()) {
      dir('build') {
        sh(label: 'Delete folders that are causing exceptions (See JENKINS-58421)', returnStatus: true,
           script: 'rm -rf ve || true; find . -type d -name vendor -exec rm -r {} \\;')
      }
    } else {
      bat(label: 'Delete ve folder', returnStatus: true,
          script: 'FOR /d /r . %%d IN ("ve") DO @IF EXIST "%%d" rmdir /s /q "%%d"')
    }
    cmd(label: 'Prepare test output', script: 'python .ci/scripts/pre_archive_test.py', returnStatus: true)
    dir('build') {
      junit(allowEmptyResults: true, keepLongStdio: true, testResults: args.testResults)
      if (args.upload) {
        tarAndUploadArtifacts(file: "test-build-artifacts-${args.id}.tgz", location: '.')
      }
    }
    if (args.upload) {
      catchError(buildResult: 'SUCCESS', message: 'Failed to archive the build test results', stageResult: 'SUCCESS') {
        def folder = cmd(label: 'Find system-tests', returnStdout: true, script: 'python .ci/scripts/search_system_tests.py').trim()
        log(level: 'INFO', text: "system-tests='${folder}'. If no empty then let's create a tarball")
        if (folder.trim()) {
          // TODO: nodeOS() should support ARM
          def os_suffix = isArm() ? 'linux' : nodeOS()
          def name = folder.replaceAll('/', '-').replaceAll('\\\\', '-').replaceAll('build', '').replaceAll('^-', '') + '-' + os_suffix
          tarAndUploadArtifacts(file: "${name}.tgz", location: folder)
        }
      }
    }
  }
}

/**
* Wrapper to tar and upload artifacts to Google Storage to avoid killing the
* disk space of the jenkins instance
*/
def tarAndUploadArtifacts(Map args = [:]) {
  def fileName = args.file.replaceAll('[^A-Za-z-0-9]','-')
  tar(file: fileName, dir: args.location, archive: false, allowMissing: true)
  googleStorageUploadExt(bucket: "gs://${JOB_GCS_BUCKET}/${env.JOB_NAME}-${env.BUILD_ID}",
                         credentialsId: "${JOB_GCS_EXT_CREDENTIALS}",
                         pattern: "${fileName}",
                         sharedPublicly: true)
}

/**
* Replace the slashes in the directory in case there are nested folders.
*/
def normalise(String directory) {
  return directory.replaceAll("[\\W]|_",'-')
}

/**
* For debugging purposes.
*/
def dumpVariables(){
  echo "### MAGE DUMP ###"
  cmd(label: 'Dump mage variables', script: 'mage dumpVariables')
  echo "### END MAGE DUMP ###"
  echo """
  ### ENV DUMP ###
  BEAT_VERSION: ${env.BEAT_VERSION}
  BEATS: ${env.BEATS}
  BUILD_DIR: ${env.BUILD_DIR}
  COMMIT_ID: ${env.COMMIT_ID}
  COVERAGE_DIR: ${env.COVERAGE_DIR}
  COVERAGE_TOOL: ${env.COVERAGE_TOOL}
  COVERAGE_TOOL_REPO: ${env.COVERAGE_TOOL_REPO}
  DOCKER_CACHE: ${env.DOCKER_CACHE}
  DOCKER_COMPOSE_PROJECT_NAME: ${env.DOCKER_COMPOSE_PROJECT_NAME}
  DOCKER_COMPOSE: ${env.DOCKER_COMPOSE}
  FIND: ${env.FIND}
  GOBUILD_FLAGS: ${env.GOBUILD_FLAGS}
  GOIMPORTS: ${env.GOIMPORTS}
  GOIMPORTS_REPO: ${env.GOIMPORTS_REPO}
  GOIMPORTS_LOCAL_PREFIX: ${env.GOIMPORTS_LOCAL_PREFIX}
  GOLINT: ${env.GOLINT}
  GOLINT_REPO: ${env.GOLINT_REPO}
  GOPACKAGES_COMMA_SEP: ${env.GOPACKAGES_COMMA_SEP}
  GOX_FLAGS: ${env.GOX_FLAGS}
  GOX_OS: ${env.GOX_OS}
  GOX_OSARCH: ${env.GOX_OSARCH}
  HOME: ${env.HOME}
  NOSETESTS_OPTIONS: ${env.NOSETESTS_OPTIONS}
  NOW: ${env.NOW}
  PATH: ${env.PATH}
  PKG_BUILD_DIR: ${env.PKG_BUILD_DIR}
  PKG_UPLOAD_DIR: ${env.PKG_UPLOAD_DIR}
  PIP_INSTALL_PARAMS: ${env.PIP_INSTALL_PARAMS}
  PROJECTS: ${env.PROJECTS}
  PROJECTS_ENV: ${env.PROJECTS_ENV}
  PYTHON_ENV: ${env.PYTHON_ENV}
  PYTHON_ENV_EXE: ${env.PYTHON_ENV_EXE}
  PYTHON_EXE: ${env.PYTHON_EXE}
  PYTHON_TEST_FILES: ${env.PYTHON_TEST_FILES}
  PROCESSES: ${env.PROCESSES}
  STRESS_TESTS: ${env.STRESS_TESTS}
  STRESS_TEST_OPTIONS: ${env.STRESS_TEST_OPTIONS}
  SYSTEM_TESTS: ${env.SYSTEM_TESTS}
  TESTIFY_TOOL_REPO: ${env.TESTIFY_TOOL_REPO}
  TEST_ENVIRONMENT: ${env.TEST_ENVIRONMENT}
  TEST_TAGS: ${env.TEST_TAGS}
  TESTING_ENVIRONMENT: ${env.TESTING_ENVIRONMENT}
  TIMEOUT: ${env.TIMEOUT}
  USERPROFILE: ${env.USERPROFILE}
  VENV_PARAMS: ${env.VENV_PARAMS}
  XPACK_SUFFIX: ${env.XPACK_SUFFIX}
  ### END ENV DUMP ###
  """
}

def isDockerInstalled(){
  if (env?.NODE_LABELS?.toLowerCase().contains('macosx')) {
    log(level: 'WARN', text: "Macosx workers require some docker-machine context. They are not used for anything related to docker stuff yet.")
    return false
  }
  if (isUnix()) {
    return sh(label: 'check for Docker', script: 'command -v docker', returnStatus: true) == 0
  }
  return false
}

/**
* Notify the build reason.
*/
def notifyBuildReason() {
  // Archive the build reason here, since the workspace can be deleted when running the parallel stages.
  archiveArtifacts(allowEmptyArchive: true, artifacts: 'build-reasons/*.*')
  if (isPR()) {
    echo 'TODO: Add a comment with the build reason (this is required to be implemented in the shared library)'
  }
}
