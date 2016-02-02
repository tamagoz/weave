FROM gliderlabs/alpine
MAINTAINER Weaveworks Inc <help@weave.works>
LABEL works.weave.role=system
RUN apk-install ca-certificates
WORKDIR /home/weave
ADD ./weaver /home/weave/
ENTRYPOINT ["/home/weave/weaver"]
