from common import *


def test_defaults():
  """Test the default settings"""
  config = """
  """

  r = helm_template(config)

  assert r["serviceaccount"][uname]["metadata"]["labels"]["app.kubernetes.io/name"] == "elastic-agent"

  c = getContainer(r)
  assert c["image"] == "docker.elastic.co/beats/elastic-agent:7.12.0-SNAPSHOT"
  assert c["imagePullPolicy"] == "IfNotPresent"

  # Default environment variables
  env_vars = [{
      'name': 'NODE_NAME',
      'valueFrom': {
        'fieldRef': {
            'fieldPath': 'spec.nodeName'
            }
        }
    }]

  for env in env_vars:
      assert env in c["env"]

  # Resources
  assert c["resources"] == {
      "requests": {"cpu": "300m", "memory": "500Mi"},
      "limits": {"memory": "1Gi"},
  }

  # Empty customizable defaults
  ts = getTemplateSpec(r)
  assert "imagePullSecrets" not in ts
  assert "tolerations" not in ts
  assert "nodeSelector" not in ts
  assert "affinity" not in ts


def test_image_settings():
  """Test the Docker image settings"""
  config = """
image:
  name: customImage
  tag: customTag
  pullPolicy: customPolicy
"""

  r = helm_template(config)
  c = getContainer(r)
  assert c["image"] == "customImage:customTag"
  assert c["imagePullPolicy"] == "customPolicy"
