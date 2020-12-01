#!/usr/bin/env groovy

@Library('apm@current') _

import groovy.transform.Field
/**
 This is required to store the stashed id with the test results to be digested with runbld
*/
@Field def stashedTestReports = [:]

pipeline {
  agent { label 'ubuntu-18 && immutable' }
  environment {
    AWS_ACCOUNT_SECRET = 'secret/observability-team/ci/elastic-observability-aws-account-auth'
    AWS_REGION = "${params.awsRegion}"
    REPO = 'beats'
    BASE_DIR = "src/github.com/elastic/${env.REPO}"
    DOCKERELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_COMPOSE_VERSION = "1.21.0"
    DOCKER_REGISTRY = 'docker.elastic.co'
    JOB_GCS_BUCKET = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    OSS_MODULE_PATTERN = '^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
    PIPELINE_LOG_LEVEL = 'INFO'
    PYTEST_ADDOPTS = "${params.PYTEST_ADDOPTS}"
    RUNBLD_DISABLE_NOTIFICATIONS = 'true'
    SLACK_CHANNEL = "#beats-build"
    TERRAFORM_VERSION = "0.12.24"
    XPACK_MODULE_PATTERN = '^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
  }
  options {
    timeout(time: 3, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
  }
  triggers {
    issueCommentTrigger('(?i)(.*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*|^/test\\W+.*$)')
  }
  parameters {
    booleanParam(name: 'allCloudTests', defaultValue: false, description: 'Run all cloud integration tests.')
    booleanParam(name: 'awsCloudTests', defaultValue: true, description: 'Run AWS cloud integration tests.')
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
        gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true)
        stashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
        dir("${BASE_DIR}"){
          // Skip all the stages except docs for PR's with asciidoc and md changes only
          setEnvVar('ONLY_DOCS', isGitRegionMatch(patterns: [ '.*\\.(asciidoc|md)' ], shouldMatchAll: true).toString())
          setEnvVar('GO_MOD_CHANGES', isGitRegionMatch(patterns: [ '^go.mod' ], shouldMatchAll: false).toString())
          setEnvVar('GO_VERSION', readFile(".go-version").trim())
          withEnv(["HOME=${env.WORKSPACE}"]) {
            retryWithSleep(retries: 2, seconds: 5){ sh(label: "Install Go ${env.GO_VERSION}", script: '.ci/scripts/install-go.sh') }
          }
        }
      }
    }
    stage('Lint'){
      options { skipDefaultCheckout() }
      environment {
        GOFLAGS = '-mod=readonly'
      }
      steps {
        withGithubNotify(context: "Lint") {
          withBeatsEnv(archive: false, id: "lint") {
            dumpVariables()
            setEnvVar('VERSION', sh(label: 'Get beat version', script: 'make get-version', returnStdout: true)?.trim())
            cmd(label: "make check-python", script: "make check-python")
            cmd(label: "make check-go", script: "make check-go")
            cmd(label: "Check for changes", script: "make check-no-changes")
          }
        }
      }
    }
    stage('Build&Test') {
      options { skipDefaultCheckout() }
      when {
        // Always when running builds on branches/tags
        // On a PR basis, skip if changes are only related to docs.
        // Always when forcing the input parameter
        anyOf {
          not { changeRequest() }                           // If no PR
          allOf {                                           // If PR and no docs changes
            expression { return env.ONLY_DOCS == "false" }
            changeRequest()
          }
          expression { return params.runAllStages }         // If UI forced
        }
      }
      steps {
        deleteDir()
        unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
        dir("${BASE_DIR}"){
          script {
            def mapParallelTasks = [:]
            def content = readYaml(file: 'Jenkinsfile.yml')
            if (content?.disabled?.when?.labels && beatsWhen(project: 'top-level', content: content?.disabled?.when)) {
              error 'Pull Request has been configured to be disabled when there is a skip-ci label match'
            } else {
              content['projects'].each { projectName ->
                generateStages(project: projectName, changeset: content['changeset']).each { k,v ->
                  mapParallelTasks["${k}"] = v
                }
              }
              notifyBuildReason()
              parallel(mapParallelTasks)
            }
          }
        }
      }
    }
    stage('Packaging') {
      agent none
      options { skipDefaultCheckout() }
      when {
        allOf {
          expression { return env.GO_MOD_CHANGES == "true" }
          changeRequest()
        }
      }
      steps {
        withGithubNotify(context: 'Packaging') {
          build(job: "Beats/packaging/${env.BRANCH_NAME}", propagate: true,  wait: true)
        }
      }
    }
  }
  post {
    success {
      writeFile(file: 'packaging.properties', text: """## To be consumed by the packaging pipeline
COMMIT=${env.GIT_BASE_COMMIT}
VERSION=${env.VERSION}-SNAPSHOT""")
      archiveArtifacts artifacts: 'packaging.properties'
    }
    always {
      deleteDir()
      unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
      runbld(stashedTestReports: stashedTestReports, project: env.REPO)
    }
    cleanup {
      // Required to enable the flaky test reporting with GitHub. Workspace exists since the post/always runs earlier
      dir("${BASE_DIR}"){
        notifyBuildResult(prComment: true,
                          slackComment: true, slackNotify: (isBranch() || isTag()),
                          analyzeFlakey: !isTag(), flakyReportIdx: "reporter-beats-beats-${getIdSuffix()}")
      }
    }
  }
}

/**
* There are only two supported branches, master and 7.x
*/
def getIdSuffix() {
  if(isPR()) {
    return getBranchIndice(env.CHANGE_TARGET)
  }
  if(isBranch()) {
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
* This method is the one used for running the parallel stages, therefore
* its arguments are passed by the beatsStages step.
*/
def generateStages(Map args = [:]) {
  def projectName = args.project
  def changeset = args.changeset
  def mapParallelStages = [:]
  def fileName = "${projectName}/Jenkinsfile.yml"
  if (fileExists(fileName)) {
    def content = readYaml(file: fileName)
    // changesetFunction argument is only required for the top-level when, stage specific when don't need it since it's an aggregation.
    if (beatsWhen(project: projectName, content: content?.when, changeset: changeset, changesetFunction: new GetProjectDependencies(steps: this))) {
      mapParallelStages = beatsStages(project: projectName, content: content, changeset: changeset, function: new RunCommand(steps: this))
    }
  } else {
    log(level: 'WARN', text: "${fileName} file does not exist. Please review the top-level Jenkinsfile.yml")
  }
  return mapParallelStages
}

def cloud(Map args = [:]) {
  node(args.label) {
    startCloudTestEnv(name: args.directory, dirs: args.dirs)
  }
  withCloudTestEnv() {
    try {
      target(context: args.context, command: args.command, directory: args.directory, label: args.label, withModule: args.withModule, isMage: true, id: args.id)
    } finally {
      terraformCleanup(name: args.directory, dir: args.directory)
    }
  }
}

def k8sTest(Map args = [:]) {
  def versions = args.versions
  versions.each{ v ->
    node(args.label) {
      stage("${args.context} ${v}"){
        withEnv(["K8S_VERSION=${v}", "KIND_VERSION=v0.7.0", "KUBECONFIG=${env.WORKSPACE}/kubecfg"]){
          withGithubNotify(context: "${args.context} ${v}") {
            withBeatsEnv(archive: false, withModule: false) {
              retryWithSleep(retries: 2, seconds: 5, backoff: true){ sh(label: "Install kind", script: ".ci/scripts/install-kind.sh") }
              retryWithSleep(retries: 2, seconds: 5, backoff: true){ sh(label: "Install kubectl", script: ".ci/scripts/install-kubectl.sh") }
              try {
                // Add some environmental resilience when setup does not work the very first time.
                def i = 0
                retryWithSleep(retries: 3, seconds: 5, backoff: true){
                  try {
                    sh(label: "Setup kind", script: ".ci/scripts/kind-setup.sh")
                  } catch(err) {
                    i++
                    sh(label: 'Delete cluster', script: 'kind delete cluster')
                    if (i > 2) {
                      error("Setup kind failed with error '${err.toString()}'")
                    }
                  }
                }
                sh(label: "Integration tests", script: "MODULE=kubernetes make -C metricbeat integration-tests")
                sh(label: "Deploy to kubernetes",script: "make -C deploy/kubernetes test")
              } finally {
                sh(label: 'Delete cluster', script: 'kind delete cluster')
              }
            }
          }
        }
      }
    }
  }
}

/**
* This method runs the given command supporting two kind of scenarios:
*  - make -C <folder> then the dir(location) is not required, aka by disaling isMage: false
*  - mage then the dir(location) is required, aka by enabling isMage: true.
*/
def target(Map args = [:]) {
  def context = args.context
  def command = args.command
  def directory = args.get('directory', '')
  def withModule = args.get('withModule', false)
  def isMage = args.get('isMage', false)
  node(args.label) {
    withGithubNotify(context: "${context}") {
      withBeatsEnv(archive: true, withModule: withModule, directory: directory, id: args.id) {
        dumpVariables()
        // make commands use -C <folder> while mage commands require the dir(folder)
        // let's support this scenario with the location variable.
        dir(isMage ? directory : '') {
          cmd(label: "${args.id?.trim() ? args.id : env.STAGE_NAME} - ${command}", script: "${command}")
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

  def goRoot, path, magefile, pythonEnv, testResults, artifacts, gox_flags

  if(isUnix()) {
    if (isArm() && is64arm()) {
      // TODO: nodeOS() should support ARM
      goRoot = "${env.WORKSPACE}/.gvm/versions/go${GO_VERSION}.linux.arm64"
      gox_flags = '-arch arm'
    } else {
      goRoot = "${env.WORKSPACE}/.gvm/versions/go${GO_VERSION}.${nodeOS()}.amd64"
      gox_flags = '-arch amd64'
    }
    path = "${env.WORKSPACE}/bin:${goRoot}/bin:${env.PATH}"
    magefile = "${WORKSPACE}/.magefile"
    pythonEnv = "${WORKSPACE}/python-env"
    testResults = '**/build/TEST*.xml'
    artifacts = '**/build/TEST*.out'
  } else {
    // NOTE: to support Windows 7 32 bits the arch in the mingw and go context paths is required.
    def mingwArch = is32() ? '32' : '64'
    def goArch = is32() ? '386' : 'amd64'
    def chocoPath = 'C:\\ProgramData\\chocolatey\\bin'
    def chocoPython3Path = 'C:\\Python38;C:\\Python38\\Scripts'
    goRoot = "${env.USERPROFILE}\\.gvm\\versions\\go${GO_VERSION}.windows.${goArch}"
    path = "${env.WORKSPACE}\\bin;${goRoot}\\bin;${chocoPath};${chocoPython3Path};C:\\tools\\mingw${mingwArch}\\bin;${env.PATH}"
    magefile = "${env.WORKSPACE}\\.magefile"
    testResults = "**\\build\\TEST*.xml"
    artifacts = "**\\build\\TEST*.out"
    gox_flags = '-arch 386'
  }

  deleteDir()
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
  // NOTE: This is required to run after the unstash
  def module = withModule ? getCommonModuleInTheChangeSet(directory) : ''
  withEnv([
    "DOCKER_PULL=0",
    "GOPATH=${env.WORKSPACE}",
    "GOROOT=${goRoot}",
    "HOME=${env.WORKSPACE}",
    "MAGEFILE_CACHE=${magefile}",
    "MODULE=${module}",
    "PATH=${path}",
    "PYTHON_ENV=${pythonEnv}",
    "RACE_DETECTOR=true",
    "TEST_COVERAGE=true",
    "TEST_TAGS=${env.TEST_TAGS},oracle",
    "GOX_FLAGS=${gox_flags}"
  ]) {
    if(isDockerInstalled()) {
      dockerLogin(secret: "${DOCKERELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
    }
    dir("${env.BASE_DIR}") {
      installTools()
      if(isUnix()) {
        // TODO (2020-04-07): This is a work-around to fix the Beat generator tests.
        // See https://github.com/elastic/beats/issues/17787.
        sh(label: 'check git config', script: '''
          if [ -z "$(git config --get user.email)" ]; then
            git config --global user.email "beatsmachine@users.noreply.github.com"
            git config --global user.name "beatsmachine"
          fi''')
      }
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
        // Tear down the setup for the permamnent workers.
        catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
          fixPermissions("${WORKSPACE}")
          deleteDir()
        }
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
    sh(label: 'Fix permissions', script: """#!/usr/bin/env bash
      set +x
      source ./dev-tools/common.bash
      docker_setup
      script/fix_permissions.sh ${location}""", returnStatus: true)
  }
}

/**
* This method installs the required dependencies that are for some reason not available in the
* CI Workers.
*/
def installTools() {
  if(isUnix()) {
    retryWithSleep(retries: 2, seconds: 5, backoff: true){ sh(label: "Install Go/Mage/Python/Docker/Terraform ${GO_VERSION}", script: '.ci/scripts/install-tools.sh') }
  } else {
    retryWithSleep(retries: 2, seconds: 5, backoff: true){ bat(label: "Install Go/Mage/Python ${GO_VERSION}", script: ".ci/scripts/install-tools.bat") }
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
    cmd(label: 'Prepare test output', script: 'python .ci/scripts/pre_archive_test.py')
    dir('build') {
      if (isUnix()) {
        cmd(label: 'Delete folders that are causing exceptions (See JENKINS-58421)',
            returnStatus: true,
            script: 'rm -rf ve || true; find . -type d -name vendor -exec rm -r {} \\;')
      } else { log(level: 'INFO', text: 'Delete folders that are causing exceptions (See JENKINS-58421) is disabled for Windows.') }
      junitAndStore(allowEmptyResults: true, keepLongStdio: true, testResults: args.testResults, stashedTestReports: stashedTestReports, id: args.id)
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
  tar(file: args.file, dir: args.location, archive: false, allowMissing: true)
  googleStorageUpload(bucket: "gs://${JOB_GCS_BUCKET}/${env.JOB_NAME}-${env.BUILD_ID}",
                      credentialsId: "${JOB_GCS_CREDENTIALS}",
                      pattern: "${args.file}",
                      sharedPublicly: true,
                      showInline: true)
}

/**
* This method executes a closure with credentials for cloud test
* environments.
*/
def withCloudTestEnv(Closure body) {
  def maskedVars = []
  def testTags = "${env.TEST_TAGS}"

  // AWS
  if (params.allCloudTests || params.awsCloudTests) {
    testTags = "${testTags},aws"
    def aws = getVaultSecret(secret: "${AWS_ACCOUNT_SECRET}").data
    if (!aws.containsKey('access_key')) {
      error("${AWS_ACCOUNT_SECRET} doesn't contain 'access_key'")
    }
    if (!aws.containsKey('secret_key')) {
      error("${AWS_ACCOUNT_SECRET} doesn't contain 'secret_key'")
    }
    maskedVars.addAll([
      [var: "AWS_REGION", password: "${env.AWS_REGION}"],
      [var: "AWS_ACCESS_KEY_ID", password: aws.access_key],
      [var: "AWS_SECRET_ACCESS_KEY", password: aws.secret_key],
    ])
  }

  withEnv([
    "TEST_TAGS=${testTags}",
  ]) {
    withEnvMask(vars: maskedVars) {
      body()
    }
  }
}

/**
* Start testing environment on cloud using terraform. Terraform files are
* stashed so they can be used by other stages. They are also archived in
* case manual cleanup is needed.
*
* Example:
*   startCloudTestEnv(name: 'x-pack-metricbeat', dirs: ['x-pack/metricbeat/module/aws'])
*   ...
*   terraformCleanup(name: 'x-pack-metricbeat', dir: 'x-pack/metricbeat')
*/
def startCloudTestEnv(Map args = [:]) {
  String name = normalise(args.name)
  def dirs = args.get('dirs',[])
  stage("${name}-prepare-cloud-env"){
    withCloudTestEnv() {
      withBeatsEnv(archive: false, withModule: false) {
        try {
          dirs?.each { folder ->
            retryWithSleep(retries: 2, seconds: 5, backoff: true){
              terraformApply(folder)
            }
          }
        } catch(err) {
          dirs?.each { folder ->
            // If it failed then cleanup without failing the build
            sh(label: 'Terraform Cleanup', script: ".ci/scripts/terraform-cleanup.sh ${folder}", returnStatus: true)
          }
        } finally {
          // Archive terraform states in case manual cleanup is needed.
          archiveArtifacts(allowEmptyArchive: true, artifacts: '**/terraform.tfstate')
        }
        stash(name: "terraform-${name}", allowEmpty: true, includes: '**/terraform.tfstate,**/.terraform/**')
      }
    }
  }
}

/**
* Run terraform in the given directory
*/
def terraformApply(String directory) {
  terraformInit(directory)
  dir(directory) {
    sh(label: "Terraform Apply on ${directory}", script: "terraform apply -auto-approve")
  }
}

/**
* Tear down the terraform environments, by looking for all terraform states in directory
* then it runs terraform destroy for each one.
* It uses terraform states previously stashed by startCloudTestEnv.
*/
def terraformCleanup(Map args = [:]) {
  String name = normalise(args.name)
  String directory = args.dir
  stage("${name}-tear-down-cloud-env"){
    withCloudTestEnv() {
      withBeatsEnv(archive: false, withModule: false) {
        unstash("terraform-${name}")
        retryWithSleep(retries: 2, seconds: 5, backoff: true) {
          sh(label: "Terraform Cleanup", script: ".ci/scripts/terraform-cleanup.sh ${directory}")
        }
      }
    }
  }
}

/**
* Prepare the terraform context in the given directory
*/
def terraformInit(String directory) {
  dir(directory) {
    sh(label: "Terraform Init on ${directory}", script: "terraform init")
  }
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
  REVIEWDOG: ${env.REVIEWDOG}
  REVIEWDOG_OPTIONS: ${env.REVIEWDOG_OPTIONS}
  REVIEWDOG_REPO: ${env.REVIEWDOG_REPO}
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
  if (isUnix()) {
    // TODO: some issues with macosx if(isInstalled(tool: 'docker', flag: '--version')) {
    return sh(label: 'check for Docker', script: 'command -v docker', returnStatus: true)
  } else {
    return false
  }
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

/**
* This class is the one used for running the parallel stages, therefore
* its arguments are passed by the beatsStages step.
*
*  What parameters/arguments are supported:
*    - label -> the worker labels
*    - project -> the name of the project that should match with the folder name.
*    - content -> the specific stage data in the <project>/Jenkinsfile.yml
*    - context -> the name of the stage, normally <project>-<stage>(-<platform>)?
*/
class RunCommand extends co.elastic.beats.BeatsFunction {
  public RunCommand(Map args = [:]){
    super(args)
  }
  public run(Map args = [:]){
    def withModule = args.content.get('withModule', false)
    if(args?.content?.containsKey('make')) {
      steps.target(context: args.context, command: args.content.make, directory: args.project, label: args.label, withModule: withModule, isMage: false, id: args.id)
    }
    if(args?.content?.containsKey('mage')) {
      steps.target(context: args.context, command: args.content.mage, directory: args.project, label: args.label, withModule: withModule, isMage: true, id: args.id)
    }
    if(args?.content?.containsKey('k8sTest')) {
      steps.k8sTest(context: args.context, versions: args.content.k8sTest.split(','), label: args.label, id: args.id)
    }
    if(args?.content?.containsKey('cloud')) {
      steps.cloud(context: args.context, command: args.content.cloud, directory: args.project, label: args.label, withModule: withModule, dirs: args.content.dirs, id: args.id)
    }
  }
}

/**
* This class retrieves the dependencies of a Go module for such it transforms them in a
* regex pattern.
*/
class GetProjectDependencies extends co.elastic.beats.BeatsFunction {
  public GetProjectDependencies(Map args = [:]){
    super(args)
  }
  public run(Map args = [:]){
    def output = ""
    steps.withEnv(["HOME=${steps.env.WORKSPACE}"]) {
      output = steps.sh(label: 'Get vendor dependency patterns', returnStdout: true,
                        script: ".ci/scripts/get-vendor-dependencies.sh ${args.project}")
    }
    return output?.split('\n').collect{ item -> item as String }
  }
}
