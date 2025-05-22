FROM scratch
COPY bin/txpool-viz /txpool-viz
ENTRYPOINT [ "./txpool-viz" ]