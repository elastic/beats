# docker_build('metricbeat-debugger-image', '.',
#     dockerfile='metricbeat/dev-tools/Dockerfile.debug')
# k8s_yaml('metricbeat/dev-tools/k8s-debug.yaml')

local_resource(
  'metricbeat-compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o metricbeat/build/metricbeat ./metricbeat',
  deps=['./metricbeat/main.go'])

load('ext://restart_process', 'docker_build_with_restart')

docker_build_with_restart(
    'metricbeat-image',
    'metricbeat',
    entrypoint=[
        '/usr/share/metricbeat/metricbeat',
        "-c", "/etc/metricbeat.yml",
        "-e",
        "-system.hostfs=/hostfs",
    ],
    dockerfile='metricbeat/k8s-dev-tools/Dockerfile.run',
    only=["build"],
    live_update=[
        sync('./metricbeat/build', '/usr/share/metricbeat'),
    ],
    )
# docker_build(
#     'metricbeat-image',
#     'metricbeat',
#     dockerfile='metricbeat/k8s-dev-tools/Dockerfile.run',
#     only=["build"]
#     )
k8s_yaml('metricbeat/k8s-dev-tools/k8s-run.yaml')
k8s_resource('metricbeat')
