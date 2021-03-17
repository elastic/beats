#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
    agent { label 'ubuntu-18 && immutable' }
    environment {
        REPO = 'beats'
        BASE_DIR = "src/github.com/elastic/${env.REPO}"
        PIPELINE_LOG_LEVEL = 'INFO'
    }
    options {
        timeout(time: 2, unit: 'HOURS')
        buildDiscarder(logRotator(numToKeepStr: '60', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
        timestamps()
        ansiColor('xterm')
        disableResume()
        durabilityHint('PERFORMANCE_OPTIMIZED')
        quietPeriod(10)
        rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    }
    stages {
        stage('Checkout') {
            options { skipDefaultCheckout() }
            steps {
                pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
                deleteDir()
                gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: false)
                stash allowEmpty: true, name: 'source', useDefaultExcludes: false
                dir("${BASE_DIR}"){
                  // Skip all the stages except docs for PR's with asciidoc and md changes only
                  setEnvVar('ONLY_DOCS', isGitRegionMatch(patterns: [ '.*\\.(asciidoc|md)' ], shouldMatchAll: true).toString())
                  setEnvVar('GO_MOD_CHANGES', isGitRegionMatch(patterns: [ '^go.mod' ], shouldMatchAll: false).toString())
                  setEnvVar('PACKAGING_CHANGES', isGitRegionMatch(patterns: [ '^dev-tools/packaging/.*' ], shouldMatchAll: false).toString())
                  setEnvVar('GO_VERSION', readFile(".go-version").trim())
                  withEnv(["HOME=${env.WORKSPACE}"]) {
                      retryWithSleep(retries: 2, seconds: 5){ sh(label: "Install Go ${env.GO_VERSION}", script: '.ci/scripts/install-go.sh') }
                  }
                }
            }
        }
        /*
        stage('Lint'){
            options { skipDefaultCheckout() }
            environment {
                GOFLAGS = '-mod=readonly'
            }
            steps {
                withMageEnv(version: env.GO_VERSION) {
                    dir("${BASE_DIR}"){
                        setEnvVar('VERSION', sh(label: 'Get beat version', script: 'make get-version', returnStdout: true)?.trim())
                        whenTrue(env.ONLY_DOCS == 'true') {
                        cmd(label: "make check", script: "make check")
                        }
                        whenTrue(env.ONLY_DOCS == 'false') {
                        cmd(label: "make check-python", script: "make check-python")
                        cmd(label: "make check-go", script: "make check-go")
                        cmd(label: "make notice", script: "make notice")
                        cmd(label: "Check for changes", script: "make check-no-changes")
                        }
                    }
                }
            }
        }*/
        stage('Build&Test') {
            options { skipDefaultCheckout() }
            steps {
                buildAndTest()
            }
        }
    }
}

def buildAndTest() {
    dir("${BASE_DIR}"){
        def mapParallelTasks = [:]
        def content = readYaml(file: 'Jenkinsfile.yml')
        content['projects'].each { projectName ->
            generateStages(project: projectName, changeset: content['changeset']).findAll {k,v -> isEnabledProject(id: k)}.each { k,v ->
                mapParallelTasks["${k}"] = v
            }
        }
        parallel(mapParallelTasks)
    }
}

def generateStages(Map args = [:]) {
    def projectName = args.project
    def changeset = args.changeset
    def mapParallelStages = [:]
    def fileName = "${projectName}/Jenkinsfile.yml"
    if (fileExists(fileName)) {
        def content = readYaml(file: fileName)
        mapParallelStages = beatsStages(project: projectName, content: content, changeset: changeset, function: new RunCommand(steps: this))
    } else {
        log(level: 'WARN', text: "${fileName} file does not exist. Please review the top-level Jenkinsfile.yml")
    }
    return mapParallelStages
}

class RunCommand extends co.elastic.beats.BeatsFunction {
    public RunCommand(Map args = [:]){
        super(args)
    }
    public run(Map args = [:]){
        if (!args.label.contains('immutable && ubuntu-18')) {
            steps.echo 'skipped for the time being. only supported linux'
            return
        }
        if(args?.content?.containsKey('mage')) {
            steps.target(context: args.context, command: args.content.mage, directory: args.project, label: args.label, isMage: true, id: args.id)
        }
    }
}

def isEnabledProject(args) {
    if (args.id.contains('arm') || args.id.contains('windows') || args.id.contains('macos')) {
        echo 'skipped for the time being. ARM/Windows/MacOS not supported'
        return false
    }
    if (args.id.contains('IntegTest')) {
        echo 'skipped for the time being. only supported unit tests'
        return false
    }
    if (args.id == 'x-pack/auditbeat-build') {
        echo 'skipped for the time being. Missing dependency in the worker -> fatal error: rpm/rpmlib.h'
        return false
    }
    if (args.id == 'filebeat-build' || args.id == 'metricbeat-unitTest') {
        echo 'skipped for the time being. can only be writable by the owner but the permissions are "-rw-rw-r--"'
        return false
    }
    if (args.id == 'journalbeat-unitTest') {
        echo 'skipped for the time being. Missing dependency in the worker -> fatal error: systemd/sd-journal.h'
        return false
    }
    if (args.id == 'heartbeat-build') {
        echo 'skipped for the time being. beat.beat.Timeout'
        return false
    }
    if (args.id == 'x-pack/packetbeat-build' || args.id == 'x-pack/packetbeat-Lint' || args.id == 'x-pack/filebeat-build' ||
        args.id == 'packetbeat-build' || args.id == 'libbeat-build') {
        echo 'skipped for the time being. Missing dependency in the worker -> fatal error: pcap.h'
        return false
    }
    return true
}

def target(Map args = [:]) {
    def command = args.command
    def context = args.context
    def directory = args.get('directory', '')
    def isMage = args.get('isMage', false)
    withNode(args.label) {
        deleteDir()
        unstash 'source'
        withMageEnv(version: env.GO_VERSION) {
            dir("${BASE_DIR}"){
                dir(isMage ? directory : '') {
                    try {
                        cmd(label: "${args.id?.trim() ? args.id : env.STAGE_NAME} - ${command}", script: "${command}")
                    } finally {
                        junit(allowEmptyResults: true, keepLongStdio: true, testResults: "build/TEST*.xml")
                    }
                }
            }
        }
    }
}

def withNode(String label, Closure body) {
    node(label) {
        body()
    }
}