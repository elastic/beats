import os
import sys

sys.path.insert(1, os.path.join(sys.path[0], "../../helpers"))
from helpers import helm_template

uname = 'release-name-elastic-agent'

def getSpec(obj):
  """Return the Spec for the pod"""
  return obj["deployment"][uname]["spec"]


def getTemplateSpec(obj):
  """Return the Pod Template Spec"""
  return obj["deployment"][uname]["spec"]["template"]["spec"]


def getContainer(obj):
  """Return the Pod container definition"""
  return obj["deployment"][uname]["spec"]["template"]["spec"]["containers"][0]
