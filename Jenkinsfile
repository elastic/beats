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
    TERRAFORM_VERSION = "0.13.7"
    XPACK_MODULE_PATTERN = '^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
    KIND_VERSION = 'v0.12.0'
    K8S_VERSION = 'v1.23.4'
  }
  options {
    timeout(time: 6, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '60', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
  }
  triggers {
    issueCommentTrigger("${obltGitHubComments()}")
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
    stage('Lint'){
      options { skipDefaultCheckout() }
      environment {
        GOFLAGS = '-mod=readonly'
      }
      steps {
        withGithubNotify(context: "Lint") {
          stageStatusCache(id: 'Lint'){
            withBeatsEnv(archive: false, id: "lint") {
              dumpVariables()
              whenTrue(env.ONLY_DOCS == 'true') {
                cmd(label: "make check", script: "make check")
              }
              whenTrue(env.ONLY_DOCS == 'false') {
                runLinting()
              }
            }
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
        runBuildAndTest(filterStage: 'mandatory')
      }
    }
    stage('Extended') {
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
        runBuildAndTest(filterStage: 'extended')
      }
    }
    stage('Packaging') {
      options { skipDefaultCheckout() }
      when {
        // On a PR basis, skip if changes are only related to docs.
        // Always when forcing the input parameter
        anyOf {
          allOf {                                           // If PR and no docs changes
            expression { return env.ONLY_DOCS == "false" }
            changeRequest()
          }
          expression { return params.runAllStages }         // If UI forced
        }
      }
      steps {
        runBuildAndTest(filterStage: 'packaging')
      }
    }
    stage('Packaging-Pipeline') {
      agent none
      options { skipDefaultCheckout() }
      when {
        allOf {
          anyOf {
            expression { return env.GO_MOD_CHANGES == "true" }
            expression { return env.PACKAGING_CHANGES == "true" }
          }
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

def runLinting() {
  def mapParallelTasks = [:]
  def content = readYaml(file: 'Jenkinsfile.yml')
  content['projects'].each { projectName ->
    generateStages(project: projectName, changeset: content['changeset'], filterStage: 'lint').each { k,v ->
      mapParallelTasks["${k}"] = v
    }
  }
  mapParallelTasks['default'] = { cmd(label: 'make check-default', script: 'make check-default') }
  mapParallelTasks['pre-commit'] = runPreCommit()
  parallel(mapParallelTasks)
}

def runPreCommit() {
  return {
    withNode(labels: 'ubuntu-18 && immutable', forceWorkspace: true){
      withGithubNotify(context: 'Check pre-commit', tab: 'tests') {
        deleteDir()
        unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
        dir("${BASE_DIR}"){
          preCommit(commit: "${GIT_BASE_COMMIT}", junit: true)
        }
      }
    }
  }
}

def runBuildAndTest(Map args = [:]) {
  def filterStage = args.get('filterStage', 'mandatory')
  deleteDir()
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
  dir("${BASE_DIR}"){
    def mapParallelTasks = [:]
    def content = readYaml(file: 'Jenkinsfile.yml')
    if (content?.disabled?.when?.labels && beatsWhen(project: 'top-level', content: content?.disabled?.when)) {
      error 'Pull Request has been configured to be disabled when there is a skip-ci label match'
    } else {
      content['projects'].each { projectName ->
        generateStages(project: projectName, changeset: content['changeset'], filterStage: filterStage).each { k,v ->
          mapParallelTasks["${k}"] = v
        }
      }
      notifyBuildReason()
      parallel(mapParallelTasks)
    }
  }
}


/**
* Only supported the main branch
*/
def getFlakyBranch() {
  if(isPR()) {
    return getBranchIndice(env.CHANGE_TARGET)
  } else {
    return getBranchIndice(env.BRANCH_NAME)
  }
}

/**
* Only supported the main branch
*/
def getBranchIndice(String compare) {
  return 'main'
}

/**
* This method is the one used for running the parallel stages, therefore
* its arguments are passed by the beatsStages step.
*/
def generateStages(Map args = [:]) {
  def projectName = args.project
  def filterStage = args.get('filterStage', 'all')
  def changeset = args.changeset
  def mapParallelStages = [:]
  def fileName = "${projectName}/Jenkinsfile.yml"
  if (fileExists(fileName)) {
    def content = readYaml(file: fileName)
    // changesetFunction argument is only required for the top-level when, stage specific when don't need it since it's an aggregation.
    if (beatsWhen(project: projectName, content: content?.when, changeset: changeset, changesetFunction: new GetProjectDependencies(steps: this))) {
      mapParallelStages = beatsStages(project: projectName, content: content, changeset: changeset, function: new RunCommand(steps: this), filterStage: filterStage)
    }
  } else {
    log(level: 'WARN', text: "${fileName} file does not exist. Please review the top-level Jenkinsfile.yml")
  }
  return mapParallelStages
}

def cloud(Map args = [:]) {
  withGithubNotify(context: args.context) {
    withNode(labels: args.label, forceWorkspace: true){
      startCloudTestEnv(name: args.directory, dirs: args.dirs, withAWS: args.withAWS)
    }
    withCloudTestEnv(args) {
      try {
        target(context: args.context, command: args.command, directory: args.directory, label: args.label, withModule: args.withModule, isMage: true, id: args.id)
      } finally {
        terraformCleanup(name: args.directory, dir: args.directory, withAWS: args.withAWS)
      }
    }
  }
}

def k8sTest(Map args = [:]) {
  def versions = args.versions
  versions.each{ v ->
    withNode(labels: args.label, forceWorkspace: true){
      stage("${args.context} ${v}"){
        withEnv(["K8S_VERSION=${v}"]){
          withGithubNotify(context: "${args.context} ${v}") {
            withBeatsEnv(archive: false, withModule: false) {
              withTools(k8s: true) {
                sh(label: "Integration tests", script: "MODULE=kubernetes make -C metricbeat integration-tests")
                sh(label: "Deploy to kubernetes",script: "make -C deploy/kubernetes test")
              }
            }
          }
        }
      }
    }
  }
}

/**
* It relies on KIND_VERSION which it's defined in the top-level environment section
*/
def withTools(Map args = [:], Closure body) {
  if (args.get('k8s', false)) {
    withEnv(["KUBECONFIG=${env.WORKSPACE}/kubecfg"]){
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
        body()
      } finally {
        sh(label: 'Delete cluster', script: 'kind delete cluster')
      }
    }
  } else {
    body()
  }
}

/**
* This method runs the packaging for ARM
*/
def packagingArm(Map args = [:]) {
  def PLATFORMS = [ 'linux/arm64' ].join(' ')
  withEnv([
    "PLATFORMS=${PLATFORMS}",
    "PACKAGES=docker"
  ]) {
    target(args)
  }
}

/**
* This method runs the packaging for Linux
*/
def packagingLinux(Map args = [:]) {
  def PLATFORMS = [ '+all',
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
  withEnv([
    "PLATFORMS=${PLATFORMS}"
  ]) {
    target(args)
  }
}

/**
* Upload the packages to their snapshot or pull request buckets
* @param beatsFolder beats folder
*/
def publishPackages(beatsFolder){
  def bucketUri = "gs://beats-ci-artifacts/snapshots"
  if (isPR()) {
    bucketUri = "gs://beats-ci-artifacts/pull-requests/pr-${env.CHANGE_ID}"
  }
  def beatsFolderName = getBeatsName(beatsFolder)
  uploadPackages("${bucketUri}/${beatsFolderName}", beatsFolder)

  // Copy those files to another location with the sha commit to test them
  // afterward.
  bucketUri = "gs://beats-ci-artifacts/commits/${env.GIT_BASE_COMMIT}"
  uploadPackages("${bucketUri}/${beatsFolderName}", beatsFolder)
}

/**
* Upload the distribution files to google cloud.
* TODO: There is a known issue with Google Storage plugin.
* @param bucketUri the buckets URI.
* @param beatsFolder the beats folder.
*/
def uploadPackages(bucketUri, beatsFolder){
  // sometimes google storage reports ResumableUploadException: 503 Server Error
  retryWithSleep(retries: 3, seconds: 5, backoff: true) {
    googleStorageUploadExt(bucket: bucketUri,
      credentialsId: "${JOB_GCS_EXT_CREDENTIALS}",
      pattern: "${beatsFolder}/build/distributions/**/*",
      sharedPublicly: true)
  }
}

/**
* Push the docker images for the given beat.
* @param beatsFolder beats folder
* @param arch what architecture
*/
def pushCIDockerImages(Map args = [:]) {
  def arch = args.get('arch', 'amd64')
  def beatsFolder = args.beatsFolder
  catchError(buildResult: 'UNSTABLE', message: 'Unable to push Docker images', stageResult: 'FAILURE') {
    if (beatsFolder.endsWith('auditbeat')) {
      tagAndPush(beatName: 'auditbeat', arch: arch)
    } else if (beatsFolder.endsWith('filebeat')) {
      tagAndPush(beatName: 'filebeat', arch: arch)
    } else if (beatsFolder.endsWith('heartbeat')) {
      tagAndPush(beatName: 'heartbeat', arch: arch)
    } else if (beatsFolder.endsWith('metricbeat')) {
      tagAndPush(beatName: 'metricbeat', arch: arch)
    } else if ("${beatsFolder}" == "packetbeat"){
      tagAndPush(beatName: 'packetbeat', arch: arch)
    } else if ("${beatsFolder}" == "x-pack/elastic-agent") {
      tagAndPush(beatName: 'elastic-agent', arch: arch)
    }
  }
}

/**
* Tag and push all the docker images for the given beat.
* @param beatName name of the Beat
*/
def tagAndPush(Map args = [:]) {
  def beatName = args.beatName
  def arch = args.get('arch', 'amd64')
  def libbetaVer = env.VERSION
  if("${env?.SNAPSHOT.trim()}" == "true"){
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
      doTagAndPush(beatName: beatName, variant: variant, sourceTag: libbetaVer, targetTag: "${tag}-${arch}", sourceNamespace: sourceNamespace)
    }
  }
}

/**
* @param beatName name of the Beat
* @param variant name of the variant used to build the docker image name
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

/**
* This method runs the end 2 end testing
*/
def e2e(Map args = [:]) {
  if (!args.e2e?.get('enabled', false)) { return }
  // Skip running the tests on branches or tags if configured.
  if (!isPR() && args.e2e?.get('when', false)) {
    if (isBranch() && !args.e2e.when.get('branches', true)) { return }
    if (isTag() && !args.e2e.when.get('tags', true)) { return }
  }
  if (args.e2e.get('entrypoint', '')?.trim()) {
    e2e_with_entrypoint(args)
  } else {
    runE2E(testMatrixFile: args.e2e?.get('testMatrixFile', ''),
           beatVersion: "${env.VERSION}-SNAPSHOT",
           gitHubCheckName: "e2e-${args.context}",
           gitHubCheckRepo: env.REPO,
           gitHubCheckSha1: env.GIT_BASE_COMMIT)
  }
}

/**
* This method runs the end 2 end testing in the same worker where the packages have been
* generated, this should help to speed up the things
*/
def e2e_with_entrypoint(Map args = [:]) {
  def entrypoint = args.e2e?.get('entrypoint')
  def dockerLogFile = "docker_logs_${entrypoint}.log"
  dir("${env.WORKSPACE}/src/github.com/elastic/e2e-testing") {
    // TBC with the target branch if running on a PR basis.
    git(branch: 'main', credentialsId: '2a9602aa-ab9f-4e52-baf3-b71ca88469c7-UserAndToken', url: 'https://github.com/elastic/e2e-testing.git')
    if(isDockerInstalled()) {
      dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
    }
    def goVersionForE2E = readFile('.go-version').trim()
    withEnv(["GO_VERSION=${goVersionForE2E}",
              "BEATS_LOCAL_PATH=${env.WORKSPACE}/${env.BASE_DIR}",
              "BEAT_VERSION=${env.VERSION}-SNAPSHOT",
              "LOG_LEVEL=TRACE"]) {
      def status = 0
      filebeat(output: dockerLogFile){
        try {
          sh(script: ".ci/scripts/${entrypoint}", label: "Run functional tests ${entrypoint}")
        } finally {
          junit(allowEmptyResults: true, keepLongStdio: true, testResults: "outputs/TEST-*.xml")
          archiveArtifacts allowEmptyArchive: true, artifacts: "outputs/TEST-*.xml"
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
  def command = args.command
  def context = args.context
  def directory = args.get('directory', '')
  def withModule = args.get('withModule', false)
  def isMage = args.get('isMage', false)
  def isE2E = args.e2e?.get('enabled', false)
  def isPackaging = args.get('package', false)
  def installK8s = args.get('installK8s', false)
  def dockerArch = args.get('dockerArch', 'amd64')
  def enableRetry = args.get('enableRetry', false)
  withNode(labels: args.label, forceWorkspace: true){
    withGithubNotify(context: "${context}") {
      withBeatsEnv(archive: true, withModule: withModule, directory: directory, id: args.id) {
        dumpVariables()
        withTools(k8s: installK8s) {
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
        }
        // Publish packages should happen always to easily consume those artifacts if the
        // e2e were triggered and failed.
        if (isPackaging) {
          publishPackages("${directory}")
          pushCIDockerImages(beatsFolder: "${directory}", arch: dockerArch)
        }
        if(isE2E) {
          e2e(args)
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
    path = "${env.WORKSPACE}/bin:${env.PATH}:/usr/local/bin"
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
            archiveArtifacts(allowEmptyArchive: true, artifacts: "${directory}/build/system-tests/docker-logs/TEST-docker-compose-*.log")
            archiveTestOutput(directory: directory, testResults: testResults, artifacts: artifacts, id: args.id, upload: upload)
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
    try {
      timeout(5) {
        sh(label: 'Fix permissions', script: """#!/usr/bin/env bash
          set +x
          echo "Cleaning up ${location}"
          source ./dev-tools/common.bash
          docker_setup
          script/fix_permissions.sh ${location}""", returnStatus: true)
      }
    } catch (Throwable e) {
      echo "There were some failures when fixing the permissions. ${e.toString()}"
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
  def directory = args.get('directory', '')

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
        withMageEnv(version: "${env.GO_VERSION}"){
          dir(directory){
            cmd(label: "Archive system tests files", script: 'mage packageSystemTests', returnStatus: true)
          }
        }

        def fileName = 'build/system-tests-*.tar.gz' // see dev-tools/mage/target/common/package.go#PackageSystemTests method
        def files = findFiles(glob: "${fileName}")

        if (files?.length > 0) {
          googleStorageUploadExt(
            bucket: "gs://${JOB_GCS_BUCKET}/${env.JOB_NAME}-${env.BUILD_ID}",
            credentialsId: "${JOB_GCS_EXT_CREDENTIALS}",
            pattern: "${fileName}",
            sharedPublicly: true
          )
        } else {
          log(level: 'WARN', text: "There are no system-tests files to upload Google Storage}")
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
* This method executes a closure with credentials for cloud test
* environments.
*/
def withCloudTestEnv(Map args = [:], Closure body) {
  def maskedVars = []
  def testTags = "${env.TEST_TAGS}"

  // Allow AWS credentials when the build was configured to do so with:
  //   - the cloudtests build parameters
  //   - the aws github label
  //   - forced with the cloud argument aws github label
  if (params.allCloudTests || params.awsCloudTests || matchesPrLabel(label: 'aws') || args.get('withAWS', false)) {
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
    log(level: 'INFO', text: 'withCloudTestEnv: it has been configured to run in AWS.')
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
    withCloudTestEnv(args) {
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
          error('startCloudTestEnv: terraform apply failed.')
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
    withCloudTestEnv(args) {
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
    steps.stageStatusCache(args){
      def withModule = args.content.get('withModule', false)
      def installK8s = args.content.get('installK8s', false)
      def withAWS = args.content.get('withAWS', false)
      //
      // What's the retry policy for fighting the flakiness:
      //   1) Lint/Packaging/Cloud/k8sTest stages don't retry, since their failures are normally legitim
      //   2) All the remaining stages will retry the command within the same worker/workspace if any failure
      //
      // NOTE: stage: lint uses target function while cloud and k8sTest use a different function
      //
      def enableRetry = (args.content.get('stage', 'enabled').toLowerCase().equals('lint') ||
                         args?.content?.containsKey('packaging-arm') ||
                         args?.content?.containsKey('packaging-linux')) ? false : true
      if(args?.content?.containsKey('make')) {
        steps.target(context: args.context,
                     command: args.content.make,
                     directory: args.project,
                     label: args.label,
                     withModule: withModule,
                     installK8s: installK8s,
                     isMage: false,
                     id: args.id,
                     enableRetry: enableRetry)
      }
      if(args?.content?.containsKey('mage')) {
        steps.target(context: args.context,
                     command: args.content.mage,
                     directory: args.project,
                     label: args.label,
                     installK8s: installK8s,
                     withModule: withModule,
                     isMage: true,
                     id: args.id,
                     enableRetry: enableRetry)
      }
      if(args?.content?.containsKey('packaging-arm')) {
        steps.packagingArm(context: args.context,
                           command: args.content.get('packaging-arm'),
                           directory: args.project,
                           label: args.label,
                           isMage: true,
                           id: args.id,
                           e2e: args.content.get('e2e'),
                           package: true,
                           dockerArch: 'arm64',
                           enableRetry: enableRetry)
      }
      if(args?.content?.containsKey('packaging-linux')) {
        steps.packagingLinux(context: args.context,
                             command: args.content.get('packaging-linux'),
                             directory: args.project,
                             label: args.label,
                             isMage: true,
                             id: args.id,
                             e2e: args.content.get('e2e'),
                             package: true,
                             dockerArch: 'amd64',
                             enableRetry: enableRetry)
      }
      if(args?.content?.containsKey('k8sTest')) {
        steps.k8sTest(context: args.context, versions: args.content.k8sTest.split(','), label: args.label, id: args.id)
      }
      if(args?.content?.containsKey('cloud')) {
        steps.cloud(context: args.context, command: args.content.cloud, directory: args.project, label: args.label, withModule: withModule, dirs: args.content.dirs, id: args.id, withAWS: withAWS)
      }
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
