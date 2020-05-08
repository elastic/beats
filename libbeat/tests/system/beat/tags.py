import os
import unittest


def tag(tag):
    """
    Decorates a test function with a tag following go build tags semantics,
    if the tag is not included in TEST_TAGS environment variable, the test is
    skipped.
    TEST_TAGS can be a comma-separated list of tags, e.g: TEST_TAGS=oracle,mssql.
    """
    def decorator(func):
        def wrapper(*args, **kwargs):
            set_tags = [
                tag.strip() for tag in os.environ.get("TEST_TAGS", "").split(",")
                if tag.strip() != ""
            ]
            if not tag in set_tags:
                raise unittest.SkipTest("tag '{}' is not included in TEST_TAGS".format(tag))
            return func(*args, **kwargs)
        wrapper.__name__ = func.__name__
        wrapper.__doc__ = func.__doc__
        return wrapper

    return decorator
