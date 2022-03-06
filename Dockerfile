FROM debian:bullseye AS photomine-base

RUN apt update && \
    apt install -y libvips42


FROM photomine-base as build

RUN apt install -y libvips-dev golang git

# Using the commit here is kind of a hack--it will force a rebuild when a new commit is made.
ARG COMMIT
ARG BRANCH
COPY ./ /workingtree/
# RUN git clone -b $BRANCH https://git.zackmarvel.com/zack/photomine.git /src && \
RUN git clone -b $BRANCH /workingtree /src && \
    cd /src && \
    go install


FROM photomine-base

COPY --from=build /root/go/bin/photomine /usr/local/bin/photomine
COPY --from=build /src/_templates /usr/local/share/photomine/_templates
