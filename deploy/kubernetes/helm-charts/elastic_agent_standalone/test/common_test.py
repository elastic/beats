from common import *


def test_adding_image_pull_secrets():
  """Test Adding pull secrets settings"""
  config = """
imagePullSecrets:
  - name: test-registry
"""
  r = helm_template(config)
  assert (
      getTemplateSpec(r)["imagePullSecrets"][0][
          "name"
      ]
      == "test-registry"
  )


def test_adding_resources_to_container():
  """Test adding resources configuration for the container"""
  config = """
resources:
  limits:
    cpu: "25m"
    memory: "128Mi"
  requests:
    cpu: "25m"
    memory: "128Mi"
"""
  r = helm_template(config)
  c = getContainer(r)

  assert c["resources"] == {
      "requests": {"cpu": "25m", "memory": "128Mi"},
      "limits": {"cpu": "25m", "memory": "128Mi"},
  }


def test_set_pod_security_context():
  """Test setting security context settings for the pod"""
  config = """
    podSecurityContext:
      fsGroup: 1001
      other: test
    """

  r = helm_template(config)
  p = getTemplateSpec(r)

  assert (p["securityContext"]["fsGroup"] == 1001)
  assert (p["securityContext"]["other"] == "test")


def test_set_container_security_context():
  config = """
    securityContext:
      runAsUser: 1001
      other: test
  """

  r = helm_template(config)
  c = getContainer(r)
  assert c["securityContext"]["runAsUser"] == 1001
  assert c["securityContext"]["other"] == "test"


def test_env_vars():
  """Test the Env vars settings"""
  config = """
extraEnvs:
  - name: 'TEST01'
    value: '01'
  - name: 'TEST02'
    value: '02'
"""

  r = helm_template(config)
  c = getContainer(r)
  env_vars = [{
      'name': 'TEST01',
      'value': '01'
    }, {
      'name': 'TEST02',
      'value': '02'
    }]

  for env in env_vars:
      assert env in c["env"]
