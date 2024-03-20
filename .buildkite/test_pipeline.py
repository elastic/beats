import pytest
import pipeline


@pytest.fixture
def ubuntu2204_aws_agent():
    return {
            "command": "fake-cmd",
            "platform": "platform-ingest-beats-ubuntu-2204-aarch64",
            "provider": "aws"
    }


def test_fetch_stage(ubuntu2204_aws_agent):
    step = pipeline.fetch_stage("test", ubuntu2204_aws_agent, "fake", "fake")
    assert step.create_entity() == {
        "label": "fake fake",
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

