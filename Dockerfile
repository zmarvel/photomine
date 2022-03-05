FROM debian:bullseye AS photomine-base

RUN apt update && \
    apt install -y libvips42


FROM photomine-base as build

RUN apt install -y libvips-dev golang git

ARG BRANCH
RUN git clone -b $BRANCH https://git.zackmarvel.com/zack/photomine.git /src && \
    cd /src && \
    go install


FROM photomine-base

COPY --from=build /root/go/bin/photomine /usr/local/bin/photomine
COPY --from=build /src/_templates /usr/local/share/photomine/_templates
