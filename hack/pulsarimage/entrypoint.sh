#!/usr/bin/env bash

export PULSAR_PREFIX_systemTopicEnabled=true
export PULSAR_PREFIX_topicLevelPoliciesEnabled=true

python3 /pulsar/bin/apply-config-from-env.py /pulsar/conf/standalone.conf

/pulsar/bin/pulsar standalone
