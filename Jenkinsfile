#!/usr/bin/env groovy

@Library('apm@current') _

import groovy.transform.Field
/**
 This is required to store the stashed id with the test results to be digested with runbld
*/
@Field def stashedTestReports = [:]

pipeline {
  agent { label 'ubuntu-18 && immutable' }
  options {
    timeout(time: 3, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '60', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    quietPeriod(10)
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
  }
  triggers {
    issueCommentTrigger('(?i)(/test)')
  }
  environment {
    A = credentials('vault-secret-id')
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
    stage('test'){
      steps {
        withCredentials([
          string(credentialsId: 'vault-addr', variable: 'VAULT_ADDR')
        ]) {
          echo "addr-OK"
        }
        withCredentials([
          string(credentialsId: 'vault-role-id', variable: 'VAULT_ROLE_ID')
        ]) {
          echo "role-ok"
        }
        withCredentials([
          string(credentialsId: 'vault-secret-id', variable: 'VAULT_SECRET_ID')
        ]) {
          echo "secret-ok"
        }
        getVaultSecret(secret: 'secret/jenkins-ci/fossa/api-token')
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult(prComment: true,
                        slackComment: true, slackNotify: (isBranch() || isTag()),
                        analyzeFlakey: !isTag())
    }
  }
}
