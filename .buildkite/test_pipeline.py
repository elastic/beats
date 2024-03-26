import os

import pytest
import pipeline


@pytest.fixture
def ubuntu2204_aws_agent():
    return {
        "command": "fake-cmd",
        "platform": "platform-ingest-beats-ubuntu-2204-aarch64",
        "provider": "aws"
    }


@pytest.fixture()
def fake_simple_group():
    return {
        "unitTest": {
            "command": "fake-cmd",
            "platform": "family/platform-ingest-beats-ubuntu-2204",
        },
        "integrationTest": {
            "command": "fake-integration",
            "platform": "family/platform-ingest-beats-ubuntu-2204",
            "env": {
                "FOO": "BAR",
            },
        },
    }


def test_fetch_stage(ubuntu2204_aws_agent):
    step = pipeline.fetch_stage("test", ubuntu2204_aws_agent, "fake", "fake-category")
    assert step.create_entity() == {
        "label": "fake test",
        "command": ["cd fake", "fake-cmd"],
        "notify": [
            {
                "github_commit_status": {
                    "context": "Fake: test",
                }
            }
        ],
        "agents": {
            "provider": "aws",
            "imagePrefix": "platform-ingest-beats-ubuntu-2204-aarch64",
            "instanceType": "t4g.large",
        },
        "artifact_paths": [
            "fake/build/*.xml",
            "fake/build/*.json",
        ],
    }


def test_fetch_group(fake_simple_group):
    group = pipeline.fetch_group(fake_simple_group, "fake-project", "testing")
    assert len(group.steps) == 2
    for step in group.steps:
        assert "testing" == step.category
        assert "gcp" == step.agent.provider

    assert group.steps[1].env.get("FOO") == "BAR"


def test_is_pr():
    os.environ["BUILDKITE_PULL_REQUEST"] = "1234"
    assert pipeline.is_pr() is True
    os.environ["BUILDKITE_PULL_REQUEST"] = "false"
    assert pipeline.is_pr() is False
