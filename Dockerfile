FROM scratch
COPY agentspec /agentspec
WORKDIR /work
ENTRYPOINT ["/agentspec"]
