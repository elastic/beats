#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'ubuntu && immutable' }
  environment {
    BASE_DIR = 'src/github.com/elastic/beats'
    GOX_FLAGS = "-arch amd64"
    DOCKER_COMPOSE_VERSION = "1.21.0"
    PIPELINE_LOG_LEVEL = "INFO"
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    disableConcurrentBuilds()
  }
  triggers {
    issueCommentTrigger('(?i).*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*')
  }
  parameters {
    booleanParam(name: 'runAllStages', defaultValue: false, description: 'Allow to run all stages.')
    booleanParam(name: 'windowsTest', defaultValue: true, description: 'Allow Windows stages.')
    booleanParam(name: 'macosTest', defaultValue: false, description: 'Allow macOS stages.')
    booleanParam(name: 'debug', defaultValue: false, description: 'Allow debug logging for Jenkins steps')
    booleanParam(name: 'dry_run', defaultValue: false, description: 'Skip build steps, it is for testing pipeline flow')
  }
  stages {
    /**
     Checkout the code and stash it, to use it on other stages.
    */
    stage('Checkout') {
      options { skipDefaultCheckout() }
      steps {
        deleteDir()
        gitCheckout(basedir: "${BASE_DIR}")
        dir("${BASE_DIR}"){
          loadConfigEnvVars()
        }
        whenTrue(params.debug){
          dumpFilteredEnvironment()
        }
        stash allowEmpty: true, name: 'source', useDefaultExcludes: false
      }
    }
    stage('Lint'){
      options { skipDefaultCheckout() }
      steps {
        makeTarget("Lint", "check")
      }
    }
    stage('Build and Test'){
      failFast false
      parallel {
        stage('Filebeat oss'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false"
            }
          }
          steps {
            makeTarget("Filebeat oss Linux", "-C filebeat testsuite")
          }
        }
        stage('Filebeat x-pack'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget("Filebeat x-pack Linux", "x-pack/filebeat", "update build test")
          }
        }
        stage('Filebeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FILEBEAT != "false" && params.macosTest
            }
          }
          steps {
            makeTarget("Filebeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C filebeat testsuite")
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
            mageTargetWin("Filebeat oss Windows Unit test", "filebeat", "unitTest")
          }
        }
        stage('Heartbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_HEARTBEAT != "false"
            }
          }
          stages {
            stage('Heartbeat oss'){
              steps {
                makeTarget("Heartbeat oss Linux", "-C heartbeat testsuite")
              }
            }
            stage('Heartbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.macosTest
                }
              }
              steps {
                makeTarget("Heartbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C heartbeat testsuite")
              }
            }
            stage('Heartbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Heartbeat oss Windows Unit test", "heartbeat", "unitTest")
              }
            }
          }
        }
        stage('Auditbeat oss'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT != "false"
            }
          }
          stages {
            stage('Auditbeat Linux'){
              steps {
                makeTarget("Auditbeat oss Linux", "-C auditbeat testsuite")
              }
            }
            stage('Auditbeat crosscompile'){
              steps {
                makeTarget("Auditbeat oss crosscompile", "-C auditbeat crosscompile")
              }
            }
            stage('Auditbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.macosTest
                }
              }
              steps {
                makeTarget("Auditbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C auditbeat testsuite")
              }
            }
            stage('Auditbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Auditbeat Windows Unit test", "auditbeat", "unitTest")
              }
            }
          }
        }
        stage('Auditbeat x-pack'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_AUDITBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget("Auditbeat x-pack Linux", "x-pack/auditbeat", "update build test")
          }
        }
        stage('Libbeat'){
          agent { label 'ubuntu && immutable' }
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
                makeTarget("Libbeat oss Linux", "-C libbeat testsuite")
              }
            }
            stage('Libbeat crosscompile'){
              steps {
                makeTarget("Libbeat oss crosscompile", "-C libbeat crosscompile")
              }
            }
            stage('Libbeat stress-tests'){
              steps {
                makeTarget("Libbeat stress-tests", "STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' -C libbeat stress-tests")
              }
            }
          }
        }
        stage('Libbeat x-pack'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_LIBBEAT_XPACK != "false"
            }
          }
          steps {
            makeTarget("Libbeat x-pack Linux", "-C x-pack/libbeat testsuite")
          }
        }
        stage('Metricbeat Unit tests'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat Unit tests", "-C metricbeat unit-tests coverage-report")
          }
        }
        stage('Metricbeat Integration tests'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat Integration tests", "-C metricbeat integration-tests-environment coverage-report")
          }
        }
        stage('Metricbeat System tests'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat System tests", "-C metricbeat update system-tests-environment coverage-report")
          }
        }
        stage('Metricbeat x-pack'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT_XPACK != "false"
            }
          }
          steps {
            mageTarget("Metricbeat x-pack Linux", "x-pack/metricbeat", "update build test")
          }
        }
        stage('Metricbeat crosscompile'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false"
            }
          }
          steps {
            makeTarget("Metricbeat oss crosscompile", "-C metricbeat crosscompile")
          }
        }
        stage('Metricbeat Mac OS X'){
          agent { label 'macosx' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_METRICBEAT != "false" && params.macosTest
            }
          }
          steps {
            makeTarget("Metricbeat oss Mac OS X", "TEST_ENVIRONMENT=0 -C metricbeat testsuite")
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
            mageTargetWin("Metricbeat Windows Unit test", "metricbeat", "unitTest")
          }
        }
        stage('Packetbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_PACKETBEAT != "false"
            }
          }
          stages {
            stage('Packetbeat oss'){
              steps {
                makeTarget("Packetbeat oss Linux", "-C packetbeat testsuite")
              }
            }
          }
        }
        stage('dockerlogbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_DOCKERLOGBEAT_XPACK != "false"
            }
          }
          stages {
            stage('Dockerlogbeat'){
              steps {
                mageTarget("Elastic Docker Logging Driver Plugin unit tests", "x-pack/dockerlogbeat", "update build test")
              }
            }
          }
        }
        stage('Winlogbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_WINLOGBEAT != "false"
            }
          }
          stages {
            stage('Winlogbeat oss'){
              steps {
                makeTarget("Winlogbeat oss crosscompile", "-C winlogbeat crosscompile")
              }
            }
            stage('Winlogbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Winlogbeat Windows Unit test", "winlogbeat", "unitTest")
              }
            }
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
            mageTargetWin("Winlogbeat Windows Unit test", "x-pack/winlogbeat", "unitTest")
          }
        }
        stage('Functionbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_FUNCTIONBEAT_XPACK != "false"
            }
          }
          stages {
            stage('Functionbeat x-pack'){
              steps {
                mageTarget("Functionbeat x-pack Linux", "x-pack/functionbeat", "update build test")
                withEnv(["GO_VERSION=1.13.1"]){
                  makeTarget("Functionbeat x-pack Linux", "-C x-pack/functionbeat test-gcp-functions")
                }
              }
            }
            stage('Functionbeat Mac OS X x-pack'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.macosTest
                }
              }
              steps {
                mageTarget("Functionbeat x-pack Mac OS X", "x-pack/functionbeat", "update build test")
              }
            }
            stage('Functionbeat Windows'){
              agent { label 'windows-immutable && windows-2019' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.windowsTest
                }
              }
              steps {
                mageTargetWin("Functionbeat Windows Unit test", "x-pack/functionbeat", "unitTest")
              }
            }
          }
        }
        stage('Journalbeat'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_JOURNALBEAT != "false"
            }
          }
          stages {
            stage('Journalbeat oss'){
              steps {
                makeTarget("Journalbeat Linux", "-C journalbeat testsuite")
              }
            }
          }
        }
        stage('Generators'){
          agent { label 'ubuntu && immutable' }
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
                makeTarget("Generators Metricbeat Linux", "-C generator/_templates/metricbeat test")
                makeTarget("Generators Metricbeat Linux", "-C generator/_templates/metricbeat test-package")
              }
            }
            stage('Generators Beat Linux'){
              steps {
                makeTarget("Generators Beat Linux", "-C generator/_templates/beat test")
                makeTarget("Generators Beat Linux", "-C generator/_templates/beat test-package")
              }
            }
            stage('Generators Metricbeat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.macosTest
                }
              }
              steps {
                makeTarget("Generators Metricbeat Mac OS X", "-C generator/_templates/metricbeat test")
              }
            }
            stage('Generators Beat Mac OS X'){
              agent { label 'macosx' }
              options { skipDefaultCheckout() }
              when {
                beforeAgent true
                expression {
                  return params.macosTest
                }
              }
              steps {
                makeTarget("Generators Beat Mac OS X", "-C generator/_templates/beat test")
              }
            }
          }
        }
        stage('Kubernetes'){
          agent { label 'ubuntu && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression {
              return env.BUILD_KUBERNETES != "false"
            }
          }
          steps {
            k8sTest(["v1.16.2","v1.15.3","v1.14.6","v1.13.10","v1.12.10","v1.11.10"])
          }
        }
      }
    }
  }
}

def makeTarget(String context, String target, boolean clean = true) {
  withGithubNotify(context: "${context}") {
    withBeatsEnv(true) {
      whenTrue(params.debug) {
        dumpFilteredEnvironment()
        dumpMage()
      }
      sh(label: "Make ${target}", script: "make ${target}")
      if (clean) {
        sh(script: 'script/fix_permissions.sh ${HOME}')
      }
    }
  }
}

def mageTarget(String context, String directory, String target) {
  withGithubNotify(context: "${context}") {
    withBeatsEnv(true) {
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

def mageTargetWin(String context, String directory, String target) {
  withGithubNotify(context: "${context}") {
    withBeatsEnvWin() {
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

def withBeatsEnv(boolean archive, Closure body) {
  def os = goos()
  def goRoot = "${env.WORKSPACE}/.gvm/versions/go${GO_VERSION}.${os}.amd64"

  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "GOROOT=${goRoot}",
    "PATH=${env.WORKSPACE}/bin:${goRoot}/bin:${env.PATH}",
    "MAGEFILE_CACHE=${WORKSPACE}/.magefile",
    "TEST_COVERAGE=true",
    "RACE_DETECTOR=true",
    "PYTHON_ENV=${WORKSPACE}/python-env",
    "TEST_TAGS=oracle",
    "DOCKER_PULL=0",
  ]) {
    deleteDir()
    unstash 'source'
    dir("${env.BASE_DIR}") {
      sh(label: "Install Go ${GO_VERSION}", script: ".ci/scripts/install-go.sh")
      sh(label: "Install docker-compose ${DOCKER_COMPOSE_VERSION}", script: ".ci/scripts/install-docker-compose.sh")
      sh(label: "Install Mage", script: "make mage")
      try {
        if(!params.dry_run){
          body()
        }
      } finally {
        if (archive) {
          catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
            junit(allowEmptyResults: true, keepLongStdio: true, testResults: "**/build/TEST*.xml")
            archiveArtifacts(allowEmptyArchive: true, artifacts: '**/build/TEST*.out')
          }
        }
        reportCoverage()
      }
    }
  }
}

def withBeatsEnvWin(Closure body) {
  final String chocoPath = 'C:\\ProgramData\\chocolatey\\bin'
  final String chocoPython3Path = 'C:\\Python38;C:\\Python38\\Scripts'
  def goRoot = "${env.USERPROFILE}\\.gvm\\versions\\go${GO_VERSION}.windows.amd64"

  withEnv([
    "HOME=${env.WORKSPACE}",
    "GOPATH=${env.WORKSPACE}",
    "GOROOT=${goRoot}",
    "PATH=${env.WORKSPACE}\\bin;${goRoot}\\bin;${chocoPath};${chocoPython3Path};${env.PATH}",
    "MAGEFILE_CACHE=${env.WORKSPACE}\\.magefile",
    "TEST_COVERAGE=true",
    "RACE_DETECTOR=true",
  ]){
    deleteDir()
    unstash 'source'
    dir("${env.BASE_DIR}"){
      bat(label: "Install Go/Mage/Python ${GO_VERSION}", script: ".ci/scripts/install-tools.bat")
      try {
        if(!params.dry_run){
          body()
        }
      } finally {
        catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
          junit(allowEmptyResults: true, keepLongStdio: true, testResults: "**\\build\\TEST*.xml")
          archiveArtifacts(allowEmptyArchive: true, artifacts: '**\\build\\TEST*.out')
        }
      }
    }
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

  throw new IllegalArgumentException("Unhandled OS name in NODE_LABELS: " + labels)
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
  echo "NOSETESTS_OPTIONS: ${env.NOSETESTS_OPTIONS}"
  echo "TEST_ENVIRONMENT: ${env.TEST_ENVIRONMENT}"
  echo "SYSTEM_TESTS: ${env.SYSTEM_TESTS}"
  echo "STRESS_TESTS: ${env.STRESS_TESTS}"
  echo "STRESS_TEST_OPTIONS: ${env.STRESS_TEST_OPTIONS}"
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
      withEnv(["K8S_VERSION=${v}"]){
        withGithubNotify(context: "K8s ${v}") {
          withBeatsEnv(false) {
            sh(label: "Install k8s", script: """
              eval "\$(gvm use ${GO_VERSION} --format=bash)"
              .ci/scripts/kind-setup.sh
            """)
            sh(label: "Kubernetes Kind",script: "make KUBECONFIG=\"\$(kind get kubeconfig-path)\" -C deploy/kubernetes test")
            sh(label: 'Delete cluster', script: 'kind delete cluster')
          }
        }
      }
    }
  }
}

def reportCoverage(){
  catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
    retry(2){
      sh(label: 'Report to Codecov', script: '''
        curl -sSLo codecov https://codecov.io/bash
        for i in auditbeat filebeat heartbeat libbeat metricbeat packetbeat winlogbeat journalbeat
        do
          FILE="${i}/build/coverage/full.cov"
          if [ -f "${FILE}" ]; then
            bash codecov -f "${FILE}"
          fi
        done
      ''')
    }
  }
}

// isChanged treats the patterns as regular expressions. In order to check if
// any file in a directoy is modified use `^<path to dir>/.*`.
def isChanged(patterns){
  return (
    params.runAllStages
    || isGitRegionMatch(patterns: patterns, comparator: 'regexp')
  )
}

def isChangedOSSCode(patterns) {
  def allPatterns = [
    "^Jenkinsfile",
    "^vendor/.*",
    "^libbeat/.*",
    "^testing/.*",
    "^dev-tools/.*",
    "^\\.ci/scripts/.*",
  ]
  allPatterns.addAll(patterns)
  allPatterns.addAll(getVendorPatterns('libbeat'))
  return isChanged(allPatterns)
}

def isChangedXPackCode(patterns) {
  def allPatterns = [
    "^Jenkinsfile",
    "^vendor/.*",
    "^libbeat/.*",
    "^dev-tools/.*",
    "^testing/.*",
    "^x-pack/libbeat/.*",
    "^\\.ci/scripts/.*",
  ]
  allPatterns.addAll(patterns)
  allPatterns.addAll(getVendorPatterns('x-pack/libbeat'))
  return isChanged(allPatterns)
}

def loadConfigEnvVars(){
  def empty = []
  env.GO_VERSION = readFile(".go-version").trim()

  // Libbeat is the core framework of Beats. It has no additional dependencies
  // on other projects in the Beats repository.
  env.BUILD_LIBBEAT = isChangedOSSCode(empty)
  env.BUILD_LIBBEAT_XPACK = isChangedXPackCode(empty)

  // Auditbeat depends on metricbeat as framework, but does not include any of
  // the modules from Metricbeat.
  // The Auditbeat x-pack build contains all functionality from OSS Auditbeat.
  env.BUILD_AUDITBEAT = isChangedOSSCode(getVendorPatterns('auditbeat'))
  env.BUILD_AUDITBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/auditbeat'))

  // Dockerlogbeat is a standalone Beat that only relies on libbeat.
  env.BUILD_DOCKERLOGBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/dockerlogbeat'))

  // Filebeat depends on libbeat only.
  // The Filebeat x-pack build contains all functionality from OSS Filebeat.
  env.BUILD_FILEBEAT = isChangedOSSCode(getVendorPatterns('filebeat'))
  env.BUILD_FILEBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/filebeat'))

  // Metricbeat depends on libbeat only.
  // The Metricbeat x-pack build contains all functionality from OSS Metricbeat.
  env.BUILD_METRICBEAT = isChangedOSSCode(getVendorPatterns('metricbeat'))
  env.BUILD_METRICBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/metricbeat'))

  // Functionbeat is a standalone beat that depends on libbeat only.
  // Functionbeat is available as x-pack build only.
  env.BUILD_FUNCTIONBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/functionbeat'))

  // Heartbeat depends on libbeat only.
  // The Heartbeat x-pack build contains all functionality from OSS Heartbeat.
  env.BUILD_HEARTBEAT = isChangedOSSCode(getVendorPatterns('heartbeat'))
  env.BUILD_HEARTBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/heartbeat'))

  // Journalbeat depends on libbeat only.
  // The Journalbeat x-pack build contains all functionality from OSS Journalbeat.
  env.BUILD_JOURNALBEAT = isChangedOSSCode(getVendorPatterns('journalbeat'))
  env.BUILD_JOURNALBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/journalbeat'))

  // Packetbeat depends on libbeat only.
  // The Packetbeat x-pack build contains all functionality from OSS Packetbeat.
  env.BUILD_PACKETBEAT = isChangedOSSCode(getVendorPatterns('packetbeat'))
  env.BUILD_PACKETBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/packetbeat'))

  // Winlogbeat depends on libbeat only.
  // The Winlogbeat x-pack build contains all functionality from OSS Winlogbeat.
  env.BUILD_WINLOGBEAT = isChangedOSSCode(getVendorPatterns('winlogbeat'))
  env.BUILD_WINLOGBEAT_XPACK = isChangedXPackCode(getVendorPatterns('x-pack/winlogbeat'))

  // The Kubernetes test use Filebeat and Metricbeat, but only need to be run
  // if the deployment scripts have been updated. No Beats specific testing is
  // involved.
  env.BUILD_KUBERNETES = isChanged(["^deploy/kubernetes/.*"])

  env.BUILD_GENERATOR = isChangedOSSCode(getVendorPatterns('generator'))
}

/**
  This method grab the dependencies of a Go module and transform them on regexp
*/
def getVendorPatterns(beatName){
  def output = ""
  docker.image("golang:${GO_VERSION}").inside{
    output = sh(label: 'Get vendor dependency patterns', returnStdout: true, script: """
      export HOME=${WORKSPACE}/${BASE_DIR}
      go list -mod=vendor -f '{{ .ImportPath }}{{ "\\n" }}{{ join .Deps "\\n" }}' ./${beatName}\
        |awk '{print \$1"/.*"}'\
        |sed -e "s#github.com/elastic/beats/v7/##g"
    """)
  }
  return output?.split('\n').collect{ item -> item as String }
}
