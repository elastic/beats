#!/usr/bin/env groovy

@Library('apm@current') _

import groovy.transform.Field


/**
  NOTE: Important note regarding the agents and labels.
  agent labels are defined in the gobld service, that's managed by infra. The required format
  is:
   - '<os-family> && immutable' for linux OS.
   - 'macosx' for the MacOS.
   - 'windows-immutable && windows-<version>' for Windows. NOTE: version might differ in some cases

  The above labels will help to set what OS family and specific version of the agent is
  required to used in the stage.
*/

/**
 This is required to store the stashed id with the test results to be digested with runbld
*/
@Field def stashedTestReports = [:]

pipeline {
  agent { label 'ubuntu-18 && immutable' }
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    GOX_FLAGS = "-arch amd64"
    DOCKER_COMPOSE_VERSION = "1.21.0"
    TERRAFORM_VERSION = "0.12.24"
    PIPELINE_LOG_LEVEL = "INFO"
    DOCKERELASTIC_SECRET = 'secret/observability-team/ci/docker-registry/prod'
    DOCKER_REGISTRY = 'docker.elastic.co'
    AWS_ACCOUNT_SECRET = 'secret/observability-team/ci/elastic-observability-aws-account-auth'
    RUNBLD_DISABLE_NOTIFICATIONS = 'true'
    JOB_GCS_BUCKET = 'beats-ci-temp'
    JOB_GCS_CREDENTIALS = 'beats-ci-gcs-plugin'
    XPACK_MODULE_PATTERN = '^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
    OSS_MODULE_PATTERN = '^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*'
    PYTEST_ADDOPTS = "${params.PYTEST_ADDOPTS}"
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
  }
  triggers {
    issueCommentTrigger('(?i)(.*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*|^/test(\\W+macos)?$)')
  }
  parameters {
    booleanParam(name: 'runAllStages', defaultValue: false, description: 'Allow to run all stages.')
    booleanParam(name: 'windowsTest', defaultValue: true, description: 'Allow Windows stages.')
    booleanParam(name: 'macosTest', defaultValue: false, description: 'Allow macOS stages.')
    booleanParam(name: 'allCloudTests', defaultValue: false, description: 'Run all cloud integration tests.')
    booleanParam(name: 'awsCloudTests', defaultValue: false, description: 'Run AWS cloud integration tests.')
    string(name: 'awsRegion', defaultValue: 'eu-central-1', description: 'Default AWS region to use for testing.')
    booleanParam(name: 'debug', defaultValue: false, description: 'Allow debug logging for Jenkins steps')
    booleanParam(name: 'dry_run', defaultValue: false, description: 'Skip build steps, it is for testing pipeline flow')
    string(name: 'PYTEST_ADDOPTS', defaultValue: '', description: 'Additional options to pass to pytest. Use PYTEST_ADDOPTS="-k pattern" to only run tests matching the specified pattern. For retries you can use `--reruns 3 --reruns-delay 15`')
  }
  stages {
    /**
     Checkout the code and stash it, to use it on other stages.
    */
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true)
        stashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
        dir("${BASE_DIR}"){
          loadConfigEnvVars()
        }
        whenTrue(params.debug){
          dumpFilteredEnvironment()
        }
      }
    }
    stage('Lint'){
      options { skipDefaultCheckout() }
      environment {
        // See https://github.com/elastic/beats/pull/19823
        GOFLAGS = '-mod=readonly'
      }
      steps {
        makeTarget(context: "Lint", target: "check")
      }
    }
    stage('Build and Test'){
      when {
        beforeAgent true
        expression { return env.ONLY_DOCS == "false" }
      }
      failFast false
      parallel {
        stage('Elastic Agent x-pack'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ELASTIC_AGENT_XPACK != "false"
            }
          }
          steps {
            mageTarget(context: "Elastic Agent x-pack Linux", directory: "x-pack/elastic-agent", target: "build test")
          }
        }
        stage('Elastic Agent x-pack Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ELASTIC_AGENT_XPACK != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin(context: "Elastic Agent x-pack Windows Unit test", directory: "x-pack/elastic-agent", target: "build unitTest")
          }
        }
        stage('Elastic Agent Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ELASTIC_AGENT_XPACK != "false" && env.BUILD_ON_MACOS != 'false'
            }
          }
          steps {
            mageTarget(context: "Elastic Agent x-pack Mac OS X", directory: "x-pack/elastic-agent", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Filebeat oss'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Filebeat oss Linux", directory: "filebeat", target: "build test", withModule: true)
          }
        }
        stage('Filebeat x-pack'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget(context: "Filebeat x-pack Linux", directory: "x-pack/filebeat", target: "build test", withModule: true)
          }
        }
        stage('Filebeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false" && env.BUILD_ON_MACOS != 'false'
            }
          }
          steps {
            mageTarget(context: "Filebeat oss Mac OS X", directory: "filebeat", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Filebeat x-pack Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT_XPACK != "false" && env.BUILD_ON_MACOS != 'false'
            }
          }
          steps {
            mageTarget(context: "Filebeat x-pack Mac OS X", directory: "x-pack/filebeat", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Filebeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin(context: "Filebeat oss Windows Unit test", directory: "filebeat", target: "build unitTest")
          }
        }
        stage('Filebeat x-pack Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT_XPACK != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin(context: "Filebeat x-pack Windows", directory: "x-pack/filebeat", target: "build unitTest")
          }
        }
        stage('Heartbeat oss'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_HEARTBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Heartbeat oss Linux", directory: "heartbeat", target: "build test")
          }
        }
        stage('Heartbeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ON_MACOS != 'false' && env.BUILD_HEARTBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Heartbeat oss Mac OS X", directory: "heartbeat", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Heartbeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return params.windowsTest &&  env.BUILD_HEARTBEAT != "false"
            }
          }
          steps {
            mageTargetWin(context: "Heartbeat oss Windows Unit test", directory: "heartbeat", target: "build unitTest")
          }
        }
        stage('Auditbeat oss Linux'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Auditbeat oss Linux", directory: "auditbeat", target: "build test")
          }
        }
        stage('Auditbeat crosscompile'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT != "false"
            }
          }
          steps {
            makeTarget(context: "Auditbeat oss crosscompile", directory: 'auditbeat', target: "crosscompile")
          }
        }
        stage('Auditbeat oss Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT != "false" && env.BUILD_ON_MACOS != 'false'
            }
          }
          steps {
            mageTarget(context: "Auditbeat oss Mac OS X", directory: "auditbeat", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Auditbeat oss Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin(context: "Auditbeat oss Windows Unit test", directory: "auditbeat", target: "build unitTest")
          }
        }
        stage('Auditbeat x-pack'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget(context: "Auditbeat x-pack Linux", directory: "x-pack/auditbeat", target: "update build test", withModule: true)
          }
        }
        stage('Auditbeat x-pack Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT_XPACK != "false" && env.BUILD_ON_MACOS != 'false'
            }
          }
          steps {
            mageTarget(context: "Auditbeat x-pack Mac OS X", directory: "x-pack/auditbeat", target: "build unitTest")
          }
        }
        stage('Auditbeat x-pack Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT_XPACK != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin(context: "Auditbeat x-pack Windows", directory: "x-pack/auditbeat", target: "build unitTest")
          }
        }
        stage('Libbeat'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_LIBBEAT != "false"
            }
          }
          stages {
            stage('Libbeat oss'){
              steps {
                mageTarget(context: "Libbeat oss Linux", directory: "libbeat", target: "build test")
              }
            }
            stage('Libbeat crosscompile'){
              steps {
                makeTarget(context: "Libbeat oss crosscompile", directory: 'libbeat', target: "crosscompile")
              }
            }
            stage('Libbeat stress-tests'){
              steps {
                makeTarget(context: "Libbeat stress-tests", target: "STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' -C libbeat stress-tests")
              }
            }
          }
        }
        stage('Libbeat x-pack'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_LIBBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget(context: "Libbeat x-pack Linux", directory: "x-pack/libbeat", target: "build test")
          }
        }
        stage('Metricbeat OSS Unit tests'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Metricbeat OSS linux/amd64 (unitTest)", directory: "metricbeat", target: "build unitTest")
          }
        }
        stage('Metricbeat OSS Go Integration tests'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Metricbeat OSS linux/amd64 (goIntegTest)", directory: "metricbeat", target: "goIntegTest", withModule: true)
          }
        }
        stage('Metricbeat OSS Python Integration tests'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Metricbeat OSS linux/amd64 (pythonIntegTest)", directory: "metricbeat", target: "pythonIntegTest", withModule: true)
          }
        }
        stage('Metricbeat x-pack'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT_XPACK != "false"
            }
          }
          stages {
            stage('Prepare cloud integration tests environments'){
              options { skipDefaultCheckout() }
              steps {
                startCloudTestEnv('x-pack-metricbeat', [
                   [cond: params.awsCloudTests, dir: 'x-pack/metricbeat/module/aws'],
                ])
              }
            }
            stage('Metricbeat x-pack'){
              options { skipDefaultCheckout() }
              steps {
                withCloudTestEnv() {
                  mageTarget(context: "Metricbeat x-pack Linux", directory: "x-pack/metricbeat", target: "build test", withModule: true)
                }
              }
            }
          }
          post {
            cleanup {
              terraformCleanup('x-pack-metricbeat', 'x-pack/metricbeat')
            }
          }
        }
        stage('Metricbeat crosscompile'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget(context: "Metricbeat OSS crosscompile", directory: 'metricbeat', target: "crosscompile")
          }
        }
        stage('Metricbeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false" && env.BUILD_ON_MACOS != 'false'
            }
          }
          steps {
            mageTarget(context: "Metricbeat OSS Mac OS X", directory: "metricbeat", target: "build unitTest")
          }
        }
        stage('Metricbeat x-pack Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT_XPACK != "false" && env.BUILD_ON_MACOS != 'false'
            }
          }
          steps {
            mageTarget(context: "Metricbeat x-pack Mac OS X", directory: "x-pack/metricbeat", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Metricbeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin(context: "Metricbeat Windows Unit test", directory: "metricbeat", target: "build unitTest")
          }
        }
        stage('Metricbeat x-pack Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT_XPACK != "false" && params.windowsTest
            }
          }
          steps {
            mageTargetWin(context: "Metricbeat x-pack Windows", directory: "x-pack/metricbeat", target: "build unitTest")
          }
        }
        stage('Packetbeat Linux'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_PACKETBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Packetbeat OSS Linux", directory: "packetbeat", target: "build test")
          }
        }
        stage('Packetbeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ON_MACOS != 'false' && env.BUILD_PACKETBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Packetbeat OSS Mac OS X", directory: "packetbeat", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Packetbeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return params.windowsTest && env.BUILD_PACKETBEAT != "false"
            }
          }
          steps {
            mageTargetWin(context: "Packetbeat OSS Windows", directory: "packetbeat", target: "build unitTest")
          }
        }
        stage('dockerlogbeat'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_DOCKERLOGBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget(context: "Elastic Docker Logging Driver Plugin unit tests", directory: "x-pack/dockerlogbeat", target: "build test")
          }
        }
        stage('Winlogbeat oss'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_WINLOGBEAT != "false"
            }
          }
          steps {
            makeTarget(context: "Winlogbeat oss crosscompile", directory: 'winlogbeat', target: "crosscompile")
          }
        }
        stage('Winlogbeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return params.windowsTest && env.BUILD_WINLOGBEAT != "false"
            }
          }
          steps {
            mageTargetWin(context: "Winlogbeat Windows Unit test", directory: "winlogbeat", target: "build unitTest")
          }
        }
        stage('Winlogbeat Windows x-pack'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return params.windowsTest && env.BUILD_WINLOGBEAT_XPACK != "false"
            }
          }
          steps {
            mageTargetWin(context: "Winlogbeat Windows Unit test", directory: "x-pack/winlogbeat", target: "build unitTest", withModule: true)
          }
        }
        stage('Functionbeat x-pack'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FUNCTIONBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget(context: "Functionbeat x-pack Linux", directory: "x-pack/functionbeat", target: "update build test")
            withEnv(["GO_VERSION=1.13.1"]){
              mageTarget(context: "Functionbeat x-pack Linux", directory: "x-pack/functionbeat", target: "testGCPFunctions")
            }
          }
        }
        stage('Functionbeat Mac OS X x-pack'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ON_MACOS != 'false' && env.BUILD_FUNCTIONBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget(context: "Functionbeat x-pack Mac OS X", directory: "x-pack/functionbeat", target: "build unitTest")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Functionbeat Windows'){
          agent { label 'windows-immutable && windows-2019' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return params.windowsTest && env.BUILD_FUNCTIONBEAT_XPACK != "false"
            }
          }
          steps {
            mageTargetWin(context: "Functionbeat Windows Unit test", directory: "x-pack/functionbeat", target: "build unitTest")
          }
        }
        stage('Journalbeat'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_JOURNALBEAT != "false"
            }
          }
          steps {
            mageTarget(context: "Journalbeat Linux", directory: "journalbeat", target: "build unitTest")
          }
        }
        stage('Generators'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_GENERATOR != "false"
            }
          }
          stages {
            stage('Generators Metricbeat Linux'){
              steps {
                makeTarget(context: "Generators Metricbeat Linux", directory: 'generator/_templates/metricbeat', target: "test")
                makeTarget(context: "Generators Metricbeat Linux", directory: 'generator/_templates/metricbeat', target: "test-package")
              }
            }
            stage('Generators Beat Linux'){
              steps {
                makeTarget(context: "Generators Beat Linux", directory: 'generator/_templates/beat', target: "test")
                makeTarget(context: "Generators Beat Linux", directory: 'generator/_templates/beat', target: "test-package")
              }
            }
          }
        }
        stage('Generators Metricbeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ON_MACOS != 'false' && env.BUILD_GENERATOR != "false"
            }
          }
          steps {
            makeTarget(context: "Generators Metricbeat Mac OS X", directory: 'generator/_templates/metricbeat', target: "test")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Generators Beat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_ON_MACOS != 'false' && env.BUILD_GENERATOR != "false"
            }
          }
          steps {
            makeTarget(context: "Generators Beat Mac OS X", directory: 'generator/_templates/beat', target: "test")
          }
          post {
            always {
              delete()
            }
          }
        }
        stage('Kubernetes'){
          agent { label 'ubuntu-18 && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_KUBERNETES != "false"
            }
          }
          steps {
            k8sTest(["v1.18.2","v1.17.2","v1.16.4","v1.15.7","v1.14.10"])
          }
        }
      }
    }
  }
  post {
    always {
      runbld()
    }
    cleanup {
      notifyBuildResult(prComment: true)
    }
  }
}

def delete() {
  dir("${env.BASE_DIR}") {
    fixPermissions("${WORKSPACE}")
  }
  deleteDir()
}

def fixPermissions(location) {
  sh(label: 'Fix permissions', script: """#!/usr/bin/env bash
    source ./dev-tools/common.bash
    docker_setup
    script/fix_permissions.sh ${location}""", returnStatus: true)
}

def makeTarget(Map args = [:]) {
  def context = args.context
  def target = args.target
  def directory = args.get('directory', '')
  def clean = args.get('clean', true)
  def withModule = args.get('withModule', false)
  def directoryFlag = directory.trim() ? "-C ${directory}" : ''
  withGithubNotify(context: "${context}") {
    withBeatsEnv(archive: true, withModule: withModule, directory: directory) {
      whenTrue(params.debug) {
        dumpFilteredEnvironment()
        dumpMage()
      }
      sh(label: "Make ${target}", script: "make ${directoryFlag} ${target}")
      whenTrue(clean) {
        fixPermissions("${HOME}")
      }
    }
  }
}

def mageTarget(Map args = [:]) {
  def context = args.context
  def directory = args.directory
  def target = args.target
  def withModule = args.get('withModule', false)
  withGithubNotify(context: "${context}") {
    withBeatsEnv(archive: true, withModule: withModule, directory: directory) {
      whenTrue(params.debug) {
        dumpFilteredEnvironment()
        dumpMage()
      }

      def verboseFlag = params.debug ? "-v" : ""
      dir(directory) {
        sh(label: "Mage ${target}", script: "mage ${verboseFlag} ${target}")
      }
    }
  }
}

def mageTargetWin(Map args = [:]) {
  def context = args.context
  def directory = args.directory
  def target = args.target
  def withModule = args.get('withModule', false)
  withGithubNotify(context: "${context}") {
    withBeatsEnvWin(withModule: withModule, directory: directory) {
      whenTrue(params.debug) {
        dumpFilteredEnvironment()
        dumpMageWin()
      }

      def verboseFlag = params.debug ? "-v" : ""
      dir(directory) {
        bat(label: "Mage ${target}", script: "mage ${verboseFlag} ${target}")
      }
    }
  }
}

def getModulePattern(String toCompare) {
  // Use contains to support the makeTarget(target: '-C <folder>') while mageTarget(directory: '<folder>')
  return (toCompare.contains('x-pack') ? env.XPACK_MODULE_PATTERN : env.OSS_MODULE_PATTERN)
}

def withBeatsEnv(Map args = [:], Closure body) {
  def archive = args.get('archive', true)
  def withModule = args.get('withModule', false)
  def directory = args.get('directory', '')
  def modulePattern
  if (withModule) {
    modulePattern = getModulePattern(directory)
  }
  def os = goos()
  def goRoot = "${env.WORKSPACE}/.gvm/versions/go${GO_VERSION}.${os}.amd64"

  deleteDir()
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")

  // NOTE: This is required to run after the unstash
  def module = withModule ? getCommonModuleInTheChangeSet(modulePattern, directory) : ''

  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "GOROOT=${goRoot}",
    "PATH=${env.WORKSPACE}/bin:${goRoot}/bin:${env.PATH}",
    "MAGEFILE_CACHE=${WORKSPACE}/.magefile",
    "TEST_COVERAGE=true",
    "RACE_DETECTOR=true",
    "PYTHON_ENV=${WORKSPACE}/python-env",
    "TEST_TAGS=${env.TEST_TAGS},oracle",
    "DOCKER_PULL=0",
    "MODULE=${module}"
  ]) {
    if(isDockerInstalled()){
      dockerLogin(secret: "${DOCKERELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
    }
    dir("${env.BASE_DIR}") {
      installTools()
      // TODO (2020-04-07): This is a work-around to fix the Beat generator tests.
      // See https://github.com/elastic/beats/issues/17787.
      setGitConfig()
      try {
        if(!params.dry_run){
          body()
        }
      } finally {
        if (archive) {
          archiveTestOutput(testResults: '**/build/TEST*.xml', artifacts: '**/build/TEST*.out')
        }
      }
    }
  }
}

/**
  This method archives and report the tests output, for such, it searches in certain folders
  to bypass some issues when working with big repositories.
*/
def archiveTestOutput(Map args = [:]) {
  catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
    if (isUnix()) {
      fixPermissions("${WORKSPACE}")
    }
    cmd(label: 'Prepare test output', script: 'python .ci/scripts/pre_archive_test.py')
    dir('build') {
      junitAndStore(allowEmptyResults: true, keepLongStdio: true, testResults: args.testResults)
      archiveArtifacts(allowEmptyArchive: true, artifacts: args.artifacts)
    }
    catchError(buildResult: 'SUCCESS', message: 'Failed to archive the build test results', stageResult: 'SUCCESS') {
      def folder = cmd(label: 'Find system-tests', returnStdout: true, script: 'python .ci/scripts/search_system_tests.py').trim()
      log(level: 'INFO', text: "system-tests='${folder}'. If no empty then let's create a tarball")
      if (folder.trim()) {
        def name = folder.replaceAll('/', '-').replaceAll('\\\\', '-').replaceAll('build', '').replaceAll('^-', '') + '-' + goos()
        tar(file: "${name}.tgz", archive: true, dir: folder)
      }
    }
  }
}

def withBeatsEnvWin(Map args = [:], Closure body) {
  def withModule = args.get('withModule', false)
  def directory = args.get('directory', '')
  def modulePattern
  if (withModule) {
    modulePattern = getModulePattern(directory)
  }
  final String chocoPath = 'C:\\ProgramData\\chocolatey\\bin'
  final String chocoPython3Path = 'C:\\Python38;C:\\Python38\\Scripts'
  def goRoot = "${env.USERPROFILE}\\.gvm\\versions\\go${GO_VERSION}.windows.amd64"

  deleteDir()
  unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")

  // NOTE: This is required to run after the unstash
  def module = withModule ? getCommonModuleInTheChangeSet(modulePattern, directory) : ''

  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "GOROOT=${goRoot}",
    "PATH=${env.WORKSPACE}\\bin;${goRoot}\\bin;${chocoPath};${chocoPython3Path};${env.PATH}",
    "MAGEFILE_CACHE=${env.WORKSPACE}\\.magefile",
    "TEST_COVERAGE=true",
    "RACE_DETECTOR=true",
    "MODULE=${module}"
  ]){
    dir("${env.BASE_DIR}"){
      installTools()
      try {
        if(!params.dry_run){
          body()
        }
      } finally {
        archiveTestOutput(testResults: "**\\build\\TEST*.xml", artifacts: "**\\build\\TEST*.out")
      }
    }
  }
}

def installTools() {
  def i = 2 // Number of retries
  if(isUnix()) {
    retryWithSleep(retries: i, seconds: 5, backoff: true){ sh(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.sh") }
    retryWithSleep(retries: i, seconds: 5, backoff: true){ sh(label: "Install docker-compose ${DOCKER_COMPOSE_VERSION}", script: ".ci/scripts/install-docker-compose.sh") }
    retryWithSleep(retries: i, seconds: 5, backoff: true){ sh(label: "Install Terraform ${TERRAFORM_VERSION}", script: ".ci/scripts/install-terraform.sh") }
    retryWithSleep(retries: i, seconds: 5, backoff: true){ sh(label: "Install Mage", script: "make mage") }
  } else {
    // Install python3 with the specific step, even though install-tools.bat will verify if it's there anyway.
    // TODO: as soon as python3 is installed in the CI Workers we will be able to remove the line below.
    retryWithSleep(retries: i, seconds: 5, backoff: true){ installTools([ [tool: 'python3', version: '3.8', exclude: 'rc'] ]) }
    retryWithSleep(retries: i, seconds: 5, backoff: true){ bat(label: "Install Go/Mage/Python ${GO_VERSION}", script: ".ci/scripts/install-tools.bat") }
  }
}

def goos(){
  def labels = env.NODE_LABELS

  if (labels.contains('linux')) {
    return 'linux'
  } else if (labels.contains('windows')) {
    return 'windows'
  } else if (labels.contains('darwin')) {
    return 'darwin'
  }

  error("Unhandled OS name in NODE_LABELS: " + labels)
}

def dumpMage(){
  echo "### MAGE DUMP ###"
  sh(label: "Dump mage variables", script: "mage dumpVariables")
  echo "### END MAGE DUMP ###"
}

def dumpMageWin(){
  echo "### MAGE DUMP ###"
  bat(label: "Dump mage variables", script: "mage dumpVariables")
  echo "### END MAGE DUMP ###"
}

def dumpFilteredEnvironment(){
  echo "### ENV DUMP ###"
  echo "PATH: ${env.PATH}"
  echo "HOME: ${env.HOME}"
  echo "USERPROFILE: ${env.USERPROFILE}"
  echo "BUILD_DIR: ${env.BUILD_DIR}"
  echo "COVERAGE_DIR: ${env.COVERAGE_DIR}"
  echo "BEATS: ${env.BEATS}"
  echo "PROJECTS: ${env.PROJECTS}"
  echo "PROJECTS_ENV: ${env.PROJECTS_ENV}"
  echo "PYTHON_ENV: ${env.PYTHON_ENV}"
  echo "PYTHON_EXE: ${env.PYTHON_EXE}"
  echo "PYTHON_ENV_EXE: ${env.PYTHON_ENV_EXE}"
  echo "VENV_PARAMS: ${env.VENV_PARAMS}"
  echo "FIND: ${env.FIND}"
  echo "GOLINT: ${env.GOLINT}"
  echo "GOLINT_REPO: ${env.GOLINT_REPO}"
  echo "REVIEWDOG: ${env.REVIEWDOG}"
  echo "REVIEWDOG_OPTIONS: ${env.REVIEWDOG_OPTIONS}"
  echo "REVIEWDOG_REPO: ${env.REVIEWDOG_REPO}"
  echo "XPACK_SUFFIX: ${env.XPACK_SUFFIX}"
  echo "PKG_BUILD_DIR: ${env.PKG_BUILD_DIR}"
  echo "PKG_UPLOAD_DIR: ${env.PKG_UPLOAD_DIR}"
  echo "COVERAGE_TOOL: ${env.COVERAGE_TOOL}"
  echo "COVERAGE_TOOL_REPO: ${env.COVERAGE_TOOL_REPO}"
  echo "TESTIFY_TOOL_REPO: ${env.TESTIFY_TOOL_REPO}"
  echo "NOW: ${env.NOW}"
  echo "GOBUILD_FLAGS: ${env.GOBUILD_FLAGS}"
  echo "GOIMPORTS: ${env.GOIMPORTS}"
  echo "GOIMPORTS_REPO: ${env.GOIMPORTS_REPO}"
  echo "GOIMPORTS_LOCAL_PREFIX: ${env.GOIMPORTS_LOCAL_PREFIX}"
  echo "PROCESSES: ${env.PROCESSES}"
  echo "TIMEOUT: ${env.TIMEOUT}"
  echo "PYTHON_TEST_FILES: ${env.PYTHON_TEST_FILES}"
  echo "PYTEST_ADDOPTS: ${env.PYTEST_ADDOPTS}"
  echo "PYTEST_OPTIONS: ${env.PYTEST_OPTIONS}"
  echo "TEST_ENVIRONMENT: ${env.TEST_ENVIRONMENT}"
  echo "SYSTEM_TESTS: ${env.SYSTEM_TESTS}"
  echo "STRESS_TESTS: ${env.STRESS_TESTS}"
  echo "STRESS_TEST_OPTIONS: ${env.STRESS_TEST_OPTIONS}"
  echo "TEST_TAGS: ${env.TEST_TAGS}"
  echo "GOX_OS: ${env.GOX_OS}"
  echo "GOX_OSARCH: ${env.GOX_OSARCH}"
  echo "GOX_FLAGS: ${env.GOX_FLAGS}"
  echo "TESTING_ENVIRONMENT: ${env.TESTING_ENVIRONMENT}"
  echo "BEAT_VERSION: ${env.BEAT_VERSION}"
  echo "COMMIT_ID: ${env.COMMIT_ID}"
  echo "DOCKER_COMPOSE_PROJECT_NAME: ${env.DOCKER_COMPOSE_PROJECT_NAME}"
  echo "DOCKER_COMPOSE: ${env.DOCKER_COMPOSE}"
  echo "DOCKER_CACHE: ${env.DOCKER_CACHE}"
  echo "GOPACKAGES_COMMA_SEP: ${env.GOPACKAGES_COMMA_SEP}"
  echo "PIP_INSTALL_PARAMS: ${env.PIP_INSTALL_PARAMS}"
  echo "### END ENV DUMP ###"
}

def k8sTest(versions){
  versions.each{ v ->
    stage("k8s ${v}"){
      withEnv(["K8S_VERSION=${v}", "KIND_VERSION=v0.7.0", "KUBECONFIG=${env.WORKSPACE}/kubecfg"]){
        withGithubNotify(context: "K8s ${v}") {
          withBeatsEnv(archive: false, withModule: false) {
            sh(label: "Install kind", script: ".ci/scripts/install-kind.sh")
            sh(label: "Install kubectl", script: ".ci/scripts/install-kubectl.sh")
            sh(label: "Setup kind", script: ".ci/scripts/kind-setup.sh")
            sh(label: "Integration tests", script: "MODULE=kubernetes make -C metricbeat integration-tests")
            sh(label: "Deploy to kubernetes",script: "make -C deploy/kubernetes test")
            sh(label: 'Delete cluster', script: 'kind delete cluster')
          }
        }
      }
    }
  }
}

/**
*  isChanged treats the patterns as regular expressions. In order to check if
*  any file in a directoy is modified use `^<path to dir>/.*`.
*
*  In addition, there are another two alternatives to report that there are
*  changes, when `runAllStages` parameter is set to true or when running on a
*  branch/tag basis.
*/
def isChanged(patterns){
  return (
    params.runAllStages   // when runAllStages UI parameter is set to true
    || !isPR()            // when running on a branch/tag
    || isGitRegionMatch(patterns: patterns, comparator: 'regexp')
  )
}

def isChangedOSSCode(patterns) {
  def allPatterns = [
    "^Jenkinsfile",
    "^go.mod",
    "^pytest.ini",
    "^libbeat/.*",
    "^testing/.*",
    "^dev-tools/.*",
    "^\\.ci/scripts/.*",
  ]
  allPatterns.addAll(patterns)
  return isChanged(allPatterns)
}

def isChangedXPackCode(patterns) {
  def allPatterns = [
    "^Jenkinsfile",
    "^go.mod",
    "^pytest.ini",
    "^libbeat/.*",
    "^dev-tools/.*",
    "^testing/.*",
    "^x-pack/libbeat/.*",
    "^\\.ci/scripts/.*",
  ]
  allPatterns.addAll(patterns)
  return isChanged(allPatterns)
}

// withCloudTestEnv executes a closure with credentials for cloud test
// environments.
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
      [var: "AWS_REGION", password: params.awsRegion],
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

def terraformInit(String directory) {
  dir(directory) {
    sh(label: "Terraform Init on ${directory}", script: "terraform init")
  }
}

def terraformApply(String directory) {
  terraformInit(directory)
  dir(directory) {
    sh(label: "Terraform Apply on ${directory}", script: "terraform apply -auto-approve")
  }
}

// Start testing environment on cloud using terraform. Terraform files are
// stashed so they can be used by other stages. They are also archived in
// case manual cleanup is needed.
//
// Example:
//   startCloudTestEnv('x-pack-metricbeat', [
//     [cond: params.awsCloudTests, dir: 'x-pack/metricbeat/module/aws'],
//   ])
//   ...
//   terraformCleanup('x-pack-metricbeat', 'x-pack/metricbeat')
def startCloudTestEnv(String name, environments = []) {
  withCloudTestEnv() {
    withBeatsEnv(archive: false, withModule: false) {
      def runAll = params.runAllCloudTests
      try {
        for (environment in environments) {
          if (environment.cond || runAll) {
            retryWithSleep(retries: 2, seconds: 5, backoff: true){
              terraformApply(environment.dir)
            }
          }
        }
      } finally {
        // Archive terraform states in case manual cleanup is needed.
        archiveArtifacts(allowEmptyArchive: true, artifacts: '**/terraform.tfstate')
      }
      stash(name: "terraform-${name}", allowEmpty: true, includes: '**/terraform.tfstate,**/.terraform/**')
    }
  }
}


// Looks for all terraform states in directory and runs terraform destroy for them,
// it uses terraform states previously stashed by startCloudTestEnv.
def terraformCleanup(String stashName, String directory) {
  stage("Remove cloud scenarios in ${directory}"){
    withCloudTestEnv() {
      withBeatsEnv(archive: false, withModule: false) {
        unstash("terraform-${stashName}")
        retryWithSleep(retries: 2, seconds: 5, backoff: true) {
          sh(label: "Terraform Cleanup", script: ".ci/scripts/terraform-cleanup.sh ${directory}")
        }
      }
    }
  }
}

def loadConfigEnvVars(){
  def empty = []
  env.GO_VERSION = readFile(".go-version").trim()

  withEnv(["HOME=${env.WORKSPACE}"]) {
    retryWithSleep(retries: 2, seconds: 5, backoff: true){ sh(label: "Install Go ${env.GO_VERSION}", script: ".ci/scripts/install-go.sh") }
  }

  // Libbeat is the core framework of Beats. It has no additional dependencies
  // on other projects in the Beats repository.
  env.BUILD_LIBBEAT = isChangedOSSCode(empty)
  env.BUILD_LIBBEAT_XPACK = isChangedXPackCode(empty)

  // Auditbeat depends on metricbeat as framework, but does not include any of
  // the modules from Metricbeat.
  // The Auditbeat x-pack build contains all functionality from OSS Auditbeat.
  env.BUILD_AUDITBEAT = isChangedOSSCode(getProjectDependencies('auditbeat'))
  env.BUILD_AUDITBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/auditbeat'))

  // Dockerlogbeat is a standalone Beat that only relies on libbeat.
  env.BUILD_DOCKERLOGBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/dockerlogbeat'))

  // Filebeat depends on libbeat only.
  // The Filebeat x-pack build contains all functionality from OSS Filebeat.
  env.BUILD_FILEBEAT = isChangedOSSCode(getProjectDependencies('filebeat'))
  env.BUILD_FILEBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/filebeat'))

  // Metricbeat depends on libbeat only.
  // The Metricbeat x-pack build contains all functionality from OSS Metricbeat.
  env.BUILD_METRICBEAT = isChangedOSSCode(getProjectDependencies('metricbeat'))
  env.BUILD_METRICBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/metricbeat'))

  // Functionbeat is a standalone beat that depends on libbeat only.
  // Functionbeat is available as x-pack build only.
  env.BUILD_FUNCTIONBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/functionbeat'))

  // Heartbeat depends on libbeat only.
  // The Heartbeat x-pack build contains all functionality from OSS Heartbeat.
  env.BUILD_HEARTBEAT = isChangedOSSCode(getProjectDependencies('heartbeat'))
  env.BUILD_HEARTBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/heartbeat'))

  // Journalbeat depends on libbeat only.
  // The Journalbeat x-pack build contains all functionality from OSS Journalbeat.
  env.BUILD_JOURNALBEAT = isChangedOSSCode(getProjectDependencies('journalbeat'))
  env.BUILD_JOURNALBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/journalbeat'))

  // Packetbeat depends on libbeat only.
  // The Packetbeat x-pack build contains all functionality from OSS Packetbeat.
  env.BUILD_PACKETBEAT = isChangedOSSCode(getProjectDependencies('packetbeat'))
  env.BUILD_PACKETBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/packetbeat'))

  // Winlogbeat depends on libbeat only.
  // The Winlogbeat x-pack build contains all functionality from OSS Winlogbeat.
  env.BUILD_WINLOGBEAT = isChangedOSSCode(getProjectDependencies('winlogbeat'))
  env.BUILD_WINLOGBEAT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/winlogbeat'))

  // Elastic-agent is a self-contained product, that depends on libbeat only.
  // The agent acts as a supervisor for other Beats like Filebeat or Metricbeat.
  // The agent is available as x-pack build only.
  env.BUILD_ELASTIC_AGENT_XPACK = isChangedXPackCode(getProjectDependencies('x-pack/elastic-agent'))

  // The Kubernetes test use Filebeat and Metricbeat, but only need to be run
  // if the deployment scripts have been updated. No Beats specific testing is
  // involved.
  env.BUILD_KUBERNETES = isChanged(["^deploy/kubernetes/.*"])

  def generatorPatterns = ['^generator/.*']
  generatorPatterns.addAll(getProjectDependencies('generator/common/beatgen'))
  generatorPatterns.addAll(getProjectDependencies('metricbeat/beater'))
  env.BUILD_GENERATOR = isChangedOSSCode(generatorPatterns)

  // Skip all the stages for changes only related to the documentation
  env.ONLY_DOCS = isDocChangedOnly()

  // Enable macOS builds when required
  env.BUILD_ON_MACOS = (params.macosTest                  // UI Input parameter is set to true
                        || !isPR()                        // For branches and tags
                        || matchesPrLabel(label: 'macOS') // If `macOS` GH label (Case-Sensitive)
                        || (env.GITHUB_COMMENT?.toLowerCase()?.contains('/test macos'))) // If `/test macos` in the GH comment (Case-Insensitive)
}

/**
  This method gathers the module name, if required, in order to run the ITs only if
  the changeset affects a specific module.

  For such, it's required to look for changes under the module folder and exclude anything else
  such as ascidoc and png files.
*/
def getCommonModuleInTheChangeSet(String pattern, String directory) {
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
  This method verifies if the changeset for the current pull request affect only changes related
  to documentation, such as asciidoc and png files.
*/
def isDocChangedOnly(){
  if (params.runAllStages || !env.CHANGE_ID?.trim()) {
    log(level: 'INFO', text: 'Speed build for docs only is disabled for branches/tags or when forcing with the runAllStages parameter.')
    return 'false'
  } else {
    log(level: "INFO", text: 'Check if the speed build for docs is enabled.')
    return isGitRegionMatch(patterns: ['.*\\.(asciidoc|png)'], shouldMatchAll: true)
  }
}

/**
  This method grab the dependencies of a Go module and transform them on regexp
*/
def getProjectDependencies(beatName){
  def os = goos()
  def goRoot = "${env.WORKSPACE}/.gvm/versions/go${GO_VERSION}.${os}.amd64"
  def output = ""

  withEnv([
    "HOME=${env.WORKSPACE}/${env.BASE_DIR}",
    "PATH=${env.WORKSPACE}/bin:${goRoot}/bin:${env.PATH}",
  ]) {
    output = sh(label: 'Get vendor dependency patterns', returnStdout: true, script: """
      go list -deps ./${beatName} \
        | grep 'elastic/beats' \
        | sed -e "s#github.com/elastic/beats/v7/##g" \
        | awk '{print "^" \$1 "/.*"}'
    """)
  }
  return output?.split('\n').collect{ item -> item as String }
}

def setGitConfig(){
  sh(label: 'check git config', script: '''
    if [ -z "$(git config --get user.email)" ]; then
      git config user.email "beatsmachine@users.noreply.github.com"
      git config user.name "beatsmachine"
    fi
  ''')
}

def isDockerInstalled(){
  return sh(label: 'check for Docker', script: 'command -v docker', returnStatus: true)
}

def junitAndStore(Map params = [:]){
  junit(params)
  // STAGE_NAME env variable could be null in some cases, so let's use the currentmilliseconds
  def stageName = env.STAGE_NAME ? env.STAGE_NAME.replaceAll("[\\W]|_",'-') : "uncategorized-${new java.util.Date().getTime()}"
  stash(includes: params.testResults, allowEmpty: true, name: stageName, useDefaultExcludes: true)
  stashedTestReports[stageName] = stageName
}

def runbld() {
  catchError(buildResult: 'SUCCESS', message: 'runbld post build action failed.') {
    if (stashedTestReports) {
      def jobName = isPR() ? 'elastic+beats+pull-request' : 'elastic+beats'
      deleteDir()
      unstashV2(name: 'source', bucket: "${JOB_GCS_BUCKET}", credentialsId: "${JOB_GCS_CREDENTIALS}")
      dir("${env.BASE_DIR}") {
        // Unstash the test reports
        stashedTestReports.each { k, v ->
          dir(k) {
            unstash(v)
          }
        }
      }
      sh(label: 'Process JUnit reports with runbld',
        script: """\
        cat >./runbld-script <<EOF
        echo "Processing JUnit reports with runbld..."
        EOF
        /usr/local/bin/runbld ./runbld-script --job-name ${jobName}
        """.stripIndent())  // stripIdent() requires '''/
    }
  }
}
