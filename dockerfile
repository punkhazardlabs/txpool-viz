FROM scratch

# Copy Go binary
COPY bin/txpool-viz /txpool-viz

# Copy frontend static files
COPY frontend/dist /frontend/dist

ENTRYPOINT ["/txpool-viz"]
