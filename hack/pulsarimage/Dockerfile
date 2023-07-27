FROM apachepulsar/pulsar:2.10.4

ENV PULSAR_PREFIX_systemTopicEnabled=true
ENV PULSAR_PREFIX_topicLevelPoliciesEnabled=true
ENV PULSAR_PREFIX_enablePackagesManagement=true
ENV PULSAR_PREFIX_zookeeperServers="localhost:2181"
RUN python3 /pulsar/bin/apply-config-from-env.py /pulsar/conf/standalone.conf

ENTRYPOINT ["/pulsar/bin/pulsar", "standalone"]
